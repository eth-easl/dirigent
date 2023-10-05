package control_plane

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/autoscaling"
	"cluster_manager/internal/control_plane/core"
	"cluster_manager/internal/control_plane/eviction_policy"
	"cluster_manager/internal/control_plane/persistence"
	placement2 "cluster_manager/internal/control_plane/placement_policy"
	"cluster_manager/pkg/synchronization"
	"cluster_manager/pkg/tracing"
	"cluster_manager/pkg/utils"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/durationpb"
	"sync"
	"sync/atomic"
	"time"
)

type ServiceInfoStorage struct {
	ServiceInfo  *proto.ServiceInfo
	ControlPlane *ControlPlane

	Controller              *autoscaling.PFStateController
	ColdStartTracingChannel *chan tracing.ColdStartLogEntry

	PlacementPolicy  placement2.PlacementPolicy
	PersistenceLayer persistence.PersistenceLayer

	NodeInformation synchronization.SyncStructure[string, core.WorkerNodeInterface]

	StartTime time.Time
}

func (ss *ServiceInfoStorage) GetAllURLs() []string {
	var res []string

	for i := 0; i < len(ss.Controller.Endpoints); i++ {
		res = append(res, ss.Controller.Endpoints[i].URL)
	}

	return res
}

// Sequential code at this point - thread safe
func (ss *ServiceInfoStorage) reconstructEndpointInController(endpoint *core.Endpoint, dpiClients synchronization.SyncStructure[string, core.DataPlaneInterface]) {
	ss.Controller.EndpointLock.Lock()
	defer ss.Controller.EndpointLock.Unlock()

	ss.Controller.Endpoints = append(ss.Controller.Endpoints, endpoint)
	atomic.AddInt64(&ss.Controller.ScalingMetadata.ActualScale, 1)

	ss.updateEndpoints(dpiClients, ss.prepareUrlList())
}

func (ss *ServiceInfoStorage) ScalingControllerLoop(nodeList synchronization.SyncStructure[string, core.WorkerNodeInterface], dpiClients synchronization.SyncStructure[string, core.DataPlaneInterface]) {
	ok := true
	desiredCount := 0

	for ok {
		desiredCount, ok = <-ss.Controller.DesiredStateChannel

		// Channel closed ==> We send the instruction to remove all endpoints
		if !ok {
			desiredCount = 0
		}

		ss.Controller.EndpointLock.Lock()

		swapped := false

		var actualScale int

		for !swapped {
			actualScale = int(ss.Controller.ScalingMetadata.ActualScale)
			swapped = atomic.CompareAndSwapInt64(&ss.Controller.ScalingMetadata.ActualScale, int64(actualScale), int64(desiredCount))
		}

		if actualScale < desiredCount {
			go ss.doUpscaling(desiredCount-actualScale, nodeList, dpiClients)
		} else if !ss.isDownScalingDisabled() && actualScale > desiredCount {
			currentState := ss.Controller.Endpoints
			toEvict := make(map[*core.Endpoint]struct{})

			for i := 0; i < actualScale-desiredCount; i++ {
				endpoint, newState := eviction_policy.EvictionPolicy(currentState)

				if _, ok := toEvict[endpoint]; ok {
					logrus.Warn("Endpoint repetition - this is a bug.")
				}
				toEvict[endpoint] = struct{}{}
				currentState = newState
			}

			if actualScale-desiredCount != len(toEvict) {
				logrus.Warn("downscaling reference error")
			}

			ss.Controller.Endpoints = ss.excludeEndpoints(ss.Controller.Endpoints, toEvict)

			go ss.doDownscaling(toEvict, ss.prepareUrlList(), dpiClients)
		}

		ss.Controller.EndpointLock.Unlock() // for all cases (>, ==, <)
	}
}

func (ss *ServiceInfoStorage) doUpscaling(toCreateCount int, nodeList synchronization.SyncStructure[string, core.WorkerNodeInterface], dpiClients synchronization.SyncStructure[string, core.DataPlaneInterface]) {
	wg := sync.WaitGroup{}

	var finalEndpoint []*core.Endpoint

	endpointMutex := sync.Mutex{}

	logrus.Debug("Need to create: ", toCreateCount, " sandboxes")

	wg.Add(toCreateCount)

	for i := 0; i < toCreateCount; i++ {
		go func() {
			defer wg.Done()

			// TODO : @Lazar, We need to ask some resources
			requested := placement2.CreateResourceMap(1, 1)
			node := placement2.ApplyPlacementPolicy(ss.PlacementPolicy, nodeList, requested)
			if node == nil {
				logrus.Warn("Failed to do placement. No nodes are schedulable.")

				atomic.AddInt64(&ss.Controller.ScalingMetadata.ActualScale, -1)
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), utils.WorkerNodeTrafficTimeout)
			defer cancel()

			resp, err := node.CreateSandbox(ctx, ss.ServiceInfo)
			if err != nil || !resp.Success {
				msg := ""
				if err != nil {
					msg = err.Error()
				}

				logrus.Warnf("Failed to start a sandbox on worker node %s (error %s)", node.GetName(), msg)

				atomic.AddInt64(&ss.Controller.ScalingMetadata.ActualScale, -1)
				return
			}

			logrus.Debug("Sandbox creation took: ", resp.LatencyBreakdown.Total.AsDuration().Milliseconds(), " ms")

			// Critical section

			logrus.Debug("Endpoint appended: ", resp.ID)

			newEndpoint := &core.Endpoint{
				SandboxID: resp.ID,
				URL:       fmt.Sprintf("%s:%d", node.GetIP(), resp.PortMappings.HostPort),
				Node:      node,
				HostPort:  resp.PortMappings.HostPort,
				CreationHistory: tracing.ColdStartLogEntry{
					ServiceName:      ss.ServiceInfo.Name,
					ContainerID:      resp.ID,
					Success:          resp.Success,
					LatencyBreakdown: resp.LatencyBreakdown,
				},
			}

			// Update worker node structure
			//workerEndpointMap, present := ss.WorkerEndpoints.AtomicGet(node.GetName())
			// TODO: Review this line François ==> Is it correct?
			// TODO: Poential race condition ==> Check this as well
			ss.NodeInformation.GetNoCheck(node.GetName()).GetEndpointMap().AtomicSet(newEndpoint, ss.ServiceInfo.Name)
			//workerEndpointMap.AtomicSet(newEndpoint, ss.ServiceInfo.Name)

			endpointMutex.Lock()
			finalEndpoint = append(finalEndpoint, newEndpoint)
			endpointMutex.Unlock()
		}()
	}

	// batch update of endpoints
	wg.Wait()

	logrus.Debug("All sandboxes have been created. Updating endpoints.")

	ss.Controller.EndpointLock.Lock()
	oldEndpointCount := len(ss.Controller.Endpoints)
	// no need for 'endpointMutex' as the barrier has already been passed
	ss.Controller.Endpoints = append(ss.Controller.Endpoints, finalEndpoint...)
	urls := ss.prepareUrlList()

	ss.Controller.EndpointLock.Unlock()

	if oldEndpointCount == len(finalEndpoint) {
		// no new updates
		return
	}

	logrus.Debug("Propagating endpoints.")
	updateEndpointTimeStart := time.Now()
	ss.updateEndpoints(dpiClients, urls)
	durationUpdateEndpoints := time.Since(updateEndpointTimeStart)
	logrus.Debug("Endpoints updated.")

	for _, endpoint := range finalEndpoint {
		endpoint.CreationHistory.LatencyBreakdown.DataplanePropagation = durationpb.New(durationUpdateEndpoints)
		*ss.ColdStartTracingChannel <- endpoint.CreationHistory
	}
}

func (ss *ServiceInfoStorage) removeEndpointFromWNStruct(e *core.Endpoint) {
	// Update worker node structure
	ss.NodeInformation.GetNoCheck(e.Node.GetName()).GetEndpointMap().Lock()
	defer ss.NodeInformation.GetNoCheck(e.Node.GetName()).GetEndpointMap().Unlock()

	ss.NodeInformation.GetNoCheck(e.Node.GetName()).GetEndpointMap().AtomicRemove(e)
}

func (ss *ServiceInfoStorage) doDownscaling(toEvict map[*core.Endpoint]struct{}, urls []*proto.EndpointInfo, dpiClients synchronization.SyncStructure[string, core.DataPlaneInterface]) {
	wg := sync.WaitGroup{}

	wg.Add(len(toEvict))

	for key := range toEvict {
		victim := key

		if victim == nil {
			logrus.Debug("Victim null - should not have happened")
			continue // why this happens?
		}

		go func(key *core.Endpoint) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), utils.WorkerNodeTrafficTimeout)
			defer cancel()

			resp, err := victim.Node.DeleteSandbox(ctx, &proto.SandboxID{
				ID:       victim.SandboxID,
				HostPort: victim.HostPort,
			})
			if err != nil || !resp.Success {
				errText := ""
				if err != nil {
					errText = err.Error()
				}
				logrus.Warnf("Failed to delete a sandbox with ID %s on worker node %s. (error : %s)", victim.SandboxID, victim.Node.GetName(), errText)
				return
			}

			ss.removeEndpointFromWNStruct(key)
		}(key)
	}

	// batch update of endpoints
	wg.Wait()

	ss.updateEndpoints(dpiClients, urls)
}

func (ss *ServiceInfoStorage) updateEndpoints(dpiClients synchronization.SyncStructure[string, core.DataPlaneInterface], endpoints []*proto.EndpointInfo) {
	wg := &sync.WaitGroup{}

	wg.Add(dpiClients.Len())

	dpiClients.Lock()
	for _, c := range dpiClients.GetMap() {
		go func(c core.DataPlaneInterface) {
			resp, err := c.UpdateEndpointList(context.Background(), &proto.DeploymentEndpointPatch{
				Service:   ss.ServiceInfo,
				Endpoints: endpoints,
			})
			if err != nil || !resp.Success {
				errText := ""
				if err != nil {
					errText = err.Error()
				}
				logrus.Warnf("Failed to update endpoint list in the data plane : %s", errText)
			}

			wg.Done()
		}(c)
	}
	dpiClients.Unlock()

	wg.Wait()
}

func (ss *ServiceInfoStorage) excludeEndpoints(total []*core.Endpoint, toExclude map[*core.Endpoint]struct{}) []*core.Endpoint {
	var result []*core.Endpoint

	for _, endpoint := range total {
		_, ok := toExclude[endpoint]

		if !ok {
			result = append(result, endpoint)
		}
	}

	return result
}

func (ss *ServiceInfoStorage) excludeSingleEndpoint(total []*core.Endpoint, toExclude *core.Endpoint) []*core.Endpoint {
	result := make([]*core.Endpoint, 0, len(total))

	for _, endpoint := range total {
		if endpoint == toExclude {
			continue
		}

		result = append(result, endpoint)
	}

	return result
}

func (ss *ServiceInfoStorage) prepareUrlList() []*proto.EndpointInfo {
	var res []*proto.EndpointInfo

	for i := 0; i < len(ss.Controller.Endpoints); i++ {
		res = append(res, &proto.EndpointInfo{
			ID:  ss.Controller.Endpoints[i].SandboxID,
			URL: ss.Controller.Endpoints[i].URL,
		})
	}

	return res
}

func (ss *ServiceInfoStorage) isDownScalingDisabled() bool {
	return time.Since(ss.StartTime) < time.Duration(ss.Controller.ScalingMetadata.AutoscalingConfig.StableWindowWidthSeconds)*time.Second // TODO: Remove hardcoded part
}
