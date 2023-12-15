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
	"os"
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

	ShouldTrace bool
	TracingFile *os.File
}

func (ss *ServiceInfoStorage) GetAllURLs() []string {
	var res []string

	for i := 0; i < len(ss.Controller.Endpoints); i++ {
		res = append(res, ss.Controller.Endpoints[i].URL)
	}

	return res
}

func (ss *ServiceInfoStorage) ScalingControllerLoop(nodeList synchronization.SyncStructure[string, core.WorkerNodeInterface], dpiClients synchronization.SyncStructure[string, core.DataPlaneInterface]) {
	isLoopRunning := true
	desiredCount := 0

	for isLoopRunning {
		desiredCount, isLoopRunning = <-ss.Controller.DesiredStateChannel
		loopStarted := time.Now()

		// Channel closed ==> We send the instruction to remove all endpoints
		if !isLoopRunning {
			desiredCount = 0
		}

		var actualScale int

		swapped := false
		for !swapped {
			actualScale = int(ss.Controller.ScalingMetadata.ActualScale)
			swapped = atomic.CompareAndSwapInt64(&ss.Controller.ScalingMetadata.ActualScale, int64(actualScale), int64(desiredCount))
		}

		ss.Controller.EndpointLock.Lock()

		if actualScale < desiredCount {
			ss.doUpscaling(desiredCount-actualScale, nodeList, dpiClients, loopStarted)
		} else if !ss.isDownScalingDisabled() && actualScale > desiredCount {
			ss.doDownscaling(actualScale, desiredCount, dpiClients)
		}

		ss.Controller.EndpointLock.Unlock() // for all cases (>, ==, <)
	}

	if ss.ShouldTrace {
		if err := ss.TracingFile.Close(); err != nil {
			logrus.Errorf("Failed to close tracing file : (%s)", err.Error())
		}
	}
}

func (ss *ServiceInfoStorage) doUpscaling(toCreateCount int, nodeList synchronization.SyncStructure[string, core.WorkerNodeInterface], dpiClients synchronization.SyncStructure[string, core.DataPlaneInterface], loopStarted time.Time) {
	wg := sync.WaitGroup{}

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
				text := ""
				if err != nil {
					text += err.Error()
				}
				logrus.Warnf("Failed to start a sandbox on worker node %s (error %s)", node.GetName(), text)

				atomic.AddInt64(&ss.Controller.ScalingMetadata.ActualScale, -1)
				return
			}

			sandboxCreationTook := time.Since(loopStarted)
			resp.LatencyBreakdown.Total = durationpb.New(sandboxCreationTook)
			logrus.Debug("Sandbox creation took: ", sandboxCreationTook.Milliseconds(), " ms")

			startEndpointPropagation := time.Now()

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
			ss.NodeInformation.AtomicGetNoCheck(node.GetName()).GetEndpointMap().AtomicSet(newEndpoint, ss.ServiceInfo.Name)

			ss.Controller.Endpoints = append(ss.Controller.Endpoints, newEndpoint)
			urls := ss.prepareEndpointInfo([]*core.Endpoint{newEndpoint}) // prepare delta for sending

			ss.updateEndpoints(dpiClients, urls)
			logrus.Debugf("Endpoint has been propagated - %s", resp.ID)

			newEndpoint.CreationHistory.LatencyBreakdown.DataplanePropagation = durationpb.New(time.Since(startEndpointPropagation))

			select {
			case *ss.ColdStartTracingChannel <- newEndpoint.CreationHistory:
			default:
			}
		}()
	}

	wg.Wait()

	if ss.ShouldTrace {
		_, _ = ss.TracingFile.WriteString(fmt.Sprint(time.Now().Unix()) + "\n")
	}
}

func (ss *ServiceInfoStorage) removeEndpointFromWNStruct(e *core.Endpoint) {
	if e.Node == nil {
		return
	}

	// Update worker node structure
	ss.NodeInformation.GetNoCheck(e.Node.GetName()).GetEndpointMap().Lock()
	defer ss.NodeInformation.GetNoCheck(e.Node.GetName()).GetEndpointMap().Unlock()

	if e.Node.GetEndpointMap() == nil {
		return
	}

	ss.NodeInformation.GetNoCheck(e.Node.GetName()).GetEndpointMap().Remove(e)
}

func (ss *ServiceInfoStorage) doDownscaling(actualScale, desiredCount int, dpiClients synchronization.SyncStructure[string, core.DataPlaneInterface]) {
	currentState := ss.Controller.Endpoints
	toEvict := make(map[*core.Endpoint]struct{})

	for i := 0; i < actualScale-desiredCount; i++ {
		endpoint, newState := eviction_policy.EvictionPolicy(currentState)
		if len(currentState) == 0 || endpoint == nil {
			logrus.Errorf("No endpoint to evict in the downscaling loop despite the actual scale is %d.", actualScale)
			continue
		}

		if _, okk := toEvict[endpoint]; okk {
			logrus.Errorf("Endpoint repetition - this is a bug.")
		}
		toEvict[endpoint] = struct{}{}
		currentState = newState
	}

	if actualScale-desiredCount != len(toEvict) {
		logrus.Errorf("downscaling reference error")
	}

	ss.excludeEndpoints(toEvict)

	///////////////////////////////////////////////////////////////////////////
	// Firstly, drain the sandboxes and remove them from the data plane(s)
	///////////////////////////////////////////////////////////////////////////
	ss.drainSandbox(dpiClients, toEvict)

	///////////////////////////////////////////////////////////////////////////
	// Secondly, kill sandboxes on the worker node(s)
	///////////////////////////////////////////////////////////////////////////
	wg := sync.WaitGroup{}
	wg.Add(len(toEvict))

	for key := range toEvict {
		if key == nil {
			logrus.Error("Victim null - should not have happened")
			continue // why this happens?
		}

		go func(victim *core.Endpoint) {
			defer wg.Done()

			deleteSandbox(victim)
			ss.removeEndpointFromWNStruct(victim)
		}(key)
	}

	// batch update of endpoints
	wg.Wait()
}

func (ss *ServiceInfoStorage) drainSandbox(dataPlanes synchronization.SyncStructure[string, core.DataPlaneInterface], toEvict map[*core.Endpoint]struct{}) {
	////////////////////////////////////////////////////////
	var toDrain []*core.Endpoint
	for elem := range toEvict {
		toDrain = append(toDrain, elem)
	}
	////////////////////////////////////////////////////////
	dataPlanes.Lock()

	wg := sync.WaitGroup{}
	wg.Add(len(dataPlanes.GetMap()))

	for _, dp := range dataPlanes.GetMap() {
		go func(dataPlane core.DataPlaneInterface) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), utils.WorkerNodeTrafficTimeout)
			defer cancel()

			_, err := dataPlane.DrainSandbox(ctx, &proto.DeploymentEndpointPatch{
				Service:   ss.ServiceInfo,
				Endpoints: ss.prepareEndpointInfo(toDrain),
			})
			if err != nil {
				logrus.Errorf("Error draining endpoints for service %s.", ss.ServiceInfo.Name)
			}
		}(dp)
	}

	dataPlanes.Unlock()
	wg.Wait()
}

func deleteSandbox(key *core.Endpoint) {
	if key.Node == nil {
		logrus.Warnf("Reference to a node on sandbox deletion not found. Ignoring request.")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), utils.WorkerNodeTrafficTimeout)
	defer cancel()

	_, err := key.Node.DeleteSandbox(ctx, &proto.SandboxID{
		ID:       key.SandboxID,
		HostPort: key.HostPort,
	})
	if err != nil {
		logrus.Warnf("Failed to delete a sandbox with ID %s on worker node %s. (error : %v)", key.SandboxID, key.Node.GetName(), err)
	}
}

func (ss *ServiceInfoStorage) updateEndpoints(dpiClients synchronization.SyncStructure[string, core.DataPlaneInterface], endpoints []*proto.EndpointInfo) {
	wg := &sync.WaitGroup{}
	wg.Add(dpiClients.Len())

	dpiClients.Lock()

	for _, dp := range dpiClients.GetMap() {
		go func(dataPlane core.DataPlaneInterface) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), utils.WorkerNodeTrafficTimeout)
			defer cancel()

			_, err := dataPlane.UpdateEndpointList(ctx, &proto.DeploymentEndpointPatch{
				Service:   ss.ServiceInfo,
				Endpoints: endpoints,
			})
			if err != nil {
				logrus.Warnf("Failed to update endpoint list in the data plane - %v", err)
			}
		}(dp)
	}

	dpiClients.Unlock()
	wg.Wait()
}

func (ss *ServiceInfoStorage) excludeEndpoints(toExclude map[*core.Endpoint]struct{}) {
	var result []*core.Endpoint

	for _, endpoint := range ss.Controller.Endpoints {
		if _, ok := toExclude[endpoint]; !ok {
			result = append(result, endpoint)
		}
	}

	ss.Controller.Endpoints = result
}

func (ss *ServiceInfoStorage) prepareEndpointInfo(endpoints []*core.Endpoint) []*proto.EndpointInfo {
	var res []*proto.EndpointInfo

	for i := 0; i < len(endpoints); i++ {
		res = append(res, &proto.EndpointInfo{
			ID:  endpoints[i].SandboxID,
			URL: endpoints[i].URL,
		})
	}

	return res
}

func (ss *ServiceInfoStorage) isDownScalingDisabled() bool {
	return time.Since(ss.StartTime) < time.Duration(ss.Controller.ScalingMetadata.AutoscalingConfig.StableWindowWidthSeconds)*time.Second // TODO: Remove hardcoded part
}
