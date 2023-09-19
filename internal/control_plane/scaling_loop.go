package control_plane

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/core"
	"cluster_manager/internal/control_plane/persistence"
	placement2 "cluster_manager/internal/control_plane/placement_policy"
	"cluster_manager/pkg/atomic_map"
	"cluster_manager/pkg/tracing"
	"cluster_manager/pkg/utils"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/durationpb"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type NodeInfoStorage struct {
	sync.Mutex

	NodeInfo map[string]*WorkerNode
}

type ServiceInfoStorage struct {
	ServiceInfo  *proto.ServiceInfo
	ControlPlane *ControlPlane

	Controller              *PFStateController
	ColdStartTracingChannel *chan tracing.ColdStartLogEntry

	PlacementPolicy  PlacementPolicy
	PersistenceLayer persistence.PersistenceLayer

	WorkerEndpoints *atomic_map.AtomicMap[string, *atomic_map.AtomicMap[*Endpoint, string]]

	StartTime time.Time
}

type Endpoint struct {
	SandboxID       string
	URL             string
	Node            core.WorkerNodeInterface
	HostPort        int32
	CreationHistory tracing.ColdStartLogEntry
}

func (ss *ServiceInfoStorage) GetAllURLs() []string {
	var res []string

	for i := 0; i < len(ss.Controller.Endpoints); i++ {
		res = append(res, ss.Controller.Endpoints[i].URL)
	}

	return res
}

func (ss *ServiceInfoStorage) reconstructEndpointInController(endpoint *Endpoint, dpiClients *atomic_map.AtomicMap[string, core.DataPlaneInterface]) {
	ss.Controller.EndpointLock.Lock()
	defer ss.Controller.EndpointLock.Unlock()

	ss.Controller.Endpoints = append(ss.Controller.Endpoints, endpoint)
	atomic.AddInt64(&ss.Controller.ScalingMetadata.ActualScale, 1)

	ss.updateEndpoints(dpiClients, ss.prepareUrlList())
}

func (ss *ServiceInfoStorage) ScalingControllerLoop(nodeList *atomic_map.AtomicMap[string, core.WorkerNodeInterface], dpiClients *atomic_map.AtomicMap[string, core.DataPlaneInterface]) {
	for {
		select {
		case desiredCount := <-*ss.Controller.DesiredStateChannel:
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
				toEvict := make(map[*Endpoint]struct{})

				for i := 0; i < actualScale-desiredCount; i++ {
					endpoint, newState := EvictionPolicy(currentState)

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
}

func (ss *ServiceInfoStorage) doUpscaling(toCreateCount int, nodeList *atomic_map.AtomicMap[string, core.WorkerNodeInterface], dpiClients *atomic_map.AtomicMap[string, core.DataPlaneInterface]) {
	wg := sync.WaitGroup{}

	var finalEndpoint []*Endpoint

	endpointMutex := sync.Mutex{}

	logrus.Debug("Need to create: ", toCreateCount, " sandboxes")

	wg.Add(toCreateCount)

	for i := 0; i < toCreateCount; i++ {
		go func() {
			defer wg.Done()

			// TODO : @Lazar, We need to ask some resources
			requested := placement2.CreateResourceMap(1, 1)
			node := ApplyPlacementPolicy(ss.PlacementPolicy, nodeList, requested)
			if node == nil {
				logrus.Warn("Failed to do placement. No nodes are schedulable.")

				atomic.AddInt64(&ss.Controller.ScalingMetadata.ActualScale, -1)
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), utils.WorkerNodeTrafficTimeout)
			defer cancel()

			resp, err := node.CreateSandbox(ctx, ss.ServiceInfo)
			if err != nil || !resp.Success {
				logrus.Warnf("Failed to start a sandbox on worker node %s (error %s)", node.GetName(), err.Error())

				atomic.AddInt64(&ss.Controller.ScalingMetadata.ActualScale, -1)
				return
			}

			logrus.Debug("Sandbox creation took: ", resp.LatencyBreakdown.Total.AsDuration().Milliseconds(), " ms")

			// Critical section
			endpointMutex.Lock()
			logrus.Debug("Endpoint appended: ", resp.ID)

			newEndpoint := &Endpoint{
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
			workerEndpointMap, present := ss.WorkerEndpoints.Get(node.GetName())
			if !present {
				logrus.Fatal("Endpoint not present in the map")
			}

			workerEndpointMap.Set(newEndpoint, ss.ServiceInfo.Name)

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

func (ss *ServiceInfoStorage) removeEndpointFromWNStruct(e *Endpoint) {
	// Update worker node structure
	worker, present := ss.WorkerEndpoints.Get(e.Node.GetName())
	if !present {
		logrus.Error("Endpoint not present in the map.")
	}
	worker.Delete(e)
}

func (ss *ServiceInfoStorage) doDownscaling(toEvict map[*Endpoint]struct{}, urls []*proto.EndpointInfo, dpiClients *atomic_map.AtomicMap[string, core.DataPlaneInterface]) {
	barrier := sync.WaitGroup{}

	barrier.Add(len(toEvict))

	for key := range toEvict {
		victim := key

		if victim == nil {
			logrus.Debug("Victim null - should not have happened")
			continue // why this happens?
		}

		go func(key *Endpoint) {
			defer barrier.Done()

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
	barrier.Wait()

	ss.updateEndpoints(dpiClients, urls)
}

func (ss *ServiceInfoStorage) updateEndpoints(dpiClients *atomic_map.AtomicMap[string, core.DataPlaneInterface], endpoints []*proto.EndpointInfo) {
	wg := &sync.WaitGroup{}

	wg.Add(dpiClients.Len())

	for _, c := range dpiClients.Values() {
		go func(c core.DataPlaneInterface) {
			resp, err := c.UpdateEndpointList(context.Background(), &proto.DeploymentEndpointPatch{
				Service:   ss.ServiceInfo,
				Endpoints: endpoints,
			})
			if err != nil || !resp.Success {
				logrus.Warn("Failed to update endpoint list in the data plane")

				apiPort, _ := strconv.ParseInt(c.GetApiPort(), 10, 64)
				proxyPort, _ := strconv.ParseInt(c.GetProxyPort(), 10, 64)

				_, err := ss.ControlPlane.DeregisterDataplane(context.Background(), &proto.DataplaneInfo{
					APIPort:   int32(apiPort),
					ProxyPort: int32(proxyPort),
				})

				if err != nil {
					logrus.Errorf("Failed to deregister dataplane : (error %s)", err.Error())
				}
			}

			wg.Done()
		}(c)
	}

	wg.Wait()
}

func (ss *ServiceInfoStorage) excludeEndpoints(total []*Endpoint, toExclude map[*Endpoint]struct{}) []*Endpoint {
	var result []*Endpoint

	for _, endpoint := range total {
		_, ok := toExclude[endpoint]

		if !ok {
			result = append(result, endpoint)
		}
	}

	return result
}

func (ss *ServiceInfoStorage) excludeSingleEndpoint(total []*Endpoint, toExclude *Endpoint) []*Endpoint {
	result := make([]*Endpoint, 0, len(total))

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
