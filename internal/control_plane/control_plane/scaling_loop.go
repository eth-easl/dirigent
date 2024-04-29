package control_plane

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/internal/control_plane/control_plane/eviction_policy"
	"cluster_manager/internal/control_plane/control_plane/per_function_state"
	"cluster_manager/internal/control_plane/control_plane/persistence"
	"cluster_manager/internal/control_plane/control_plane/placement_policy"
	"cluster_manager/pkg/synchronization"
	"cluster_manager/pkg/tracing"
	"cluster_manager/pkg/utils"
	"cluster_manager/proto"
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

	Controller              *per_function_state.PFState
	ColdStartTracingChannel chan tracing.ColdStartLogEntry

	PlacementPolicy  placement_policy.PlacementPolicy
	PersistenceLayer persistence.PersistenceLayer

	DataPlaneConnections synchronization.SyncStructure[string, core.DataPlaneInterface]
	NIStorage            synchronization.SyncStructure[string, core.WorkerNodeInterface]

	StartTime time.Time
}

func (ss *ServiceInfoStorage) GetAllURLs() []string {
	var res []string

	for i := 0; i < len(ss.Controller.Endpoints); i++ {
		res = append(res, ss.Controller.Endpoints[i].URL)
	}

	return res
}

func (ss *ServiceInfoStorage) ScalingControllerLoop() {
	isLoopRunning := true
	desiredCount := 0

	for isLoopRunning {
		desiredCount, isLoopRunning = <-ss.Controller.DesiredStateChannel
		loopStarted := time.Now()

		lastValue := false
		for !lastValue {
			select {
			case desiredCount, isLoopRunning = <-ss.Controller.DesiredStateChannel:
				lastValue = false
			default:
				lastValue = true
			}
		}

		var actualScale int

		swapped := false
		for !swapped {
			actualScale = int(ss.Controller.ScalingMetadata.ActualScale)
			swapped = atomic.CompareAndSwapInt64(&ss.Controller.ScalingMetadata.ActualScale, int64(actualScale), int64(desiredCount))
		}

		// Channel closed ==> We send the instruction to remove all endpoints
		if !isLoopRunning {
			desiredCount = 0

			ss.Controller.EndpointLock.Lock()
			ss.doDownscaling(actualScale, desiredCount)
			ss.Controller.EndpointLock.Unlock()
			break
		}

		ss.Controller.EndpointLock.Lock()

		if actualScale < desiredCount {
			ss.doUpscaling(desiredCount-actualScale, loopStarted)
		} else if !ss.isDownScalingDisabled() && actualScale > desiredCount {
			ss.doDownscaling(actualScale, desiredCount)
		}

		ss.Controller.EndpointLock.Unlock() // for all cases (>, ==, <)
	}
}

func (ss *ServiceInfoStorage) doUpscaling(toCreateCount int, loopStarted time.Time) {
	wg := sync.WaitGroup{}

	logrus.Debug("Need to create: ", toCreateCount, " sandboxes")

	wg.Add(toCreateCount)

	for i := 0; i < toCreateCount; i++ {
		go func() {
			defer wg.Done()

			requested := placement_policy.CreateResourceMap(ss.ServiceInfo.GetRequestedCpu(), ss.ServiceInfo.GetRequestedMemory())
			node := placement_policy.ApplyPlacementPolicy(ss.PlacementPolicy, ss.NIStorage, requested)
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

			costPersistStart := time.Now()

			if ss.ControlPlane.Config.EndpointPersistence {
				ss.ControlPlane.PersistenceLayer.StoreEndpoint(context.Background(), core.Endpoint{
					SandboxID: resp.ID,
					URL:       fmt.Sprintf("%s:%d", node.GetIP(), resp.PortMappings.HostPort),
					Node:      node,
					HostPort:  resp.PortMappings.HostPort,
				})
			}

			newEndpoint := &core.Endpoint{
				SandboxID: resp.ID,
				URL:       fmt.Sprintf("%s:%d", node.GetIP(), resp.PortMappings.HostPort),
				Node:      node,
				HostPort:  resp.PortMappings.HostPort,
				CreationHistory: tracing.ColdStartLogEntry{
					ServiceName:      ss.ServiceInfo.Name,
					ContainerID:      resp.ID,
					Success:          resp.Success,
					PersistenceCost:  time.Since(costPersistStart),
					LatencyBreakdown: resp.LatencyBreakdown,
				},
			}

			startEndpointPropagation := time.Now()

			// Update worker node structure
			ss.NIStorage.AtomicGetNoCheck(node.GetName()).GetEndpointMap().AtomicSet(newEndpoint, ss.ServiceInfo.Name)

			ss.Controller.Endpoints = append(ss.Controller.Endpoints, newEndpoint)
			urls := ss.prepareEndpointInfo([]*core.Endpoint{newEndpoint}) // prepare delta for sending

			ss.updateEndpoints(urls)
			logrus.Debugf("Endpoint has been propagated - %s", resp.ID)

			newEndpoint.CreationHistory.LatencyBreakdown.DataplanePropagation = durationpb.New(time.Since(startEndpointPropagation))

			ss.ColdStartTracingChannel <- newEndpoint.CreationHistory
		}()
	}

	wg.Wait()
}

func (ss *ServiceInfoStorage) removeEndpointFromWNStruct(e *core.Endpoint) {
	if e.Node == nil {
		return
	}

	// Update worker node structure
	ss.NIStorage.GetNoCheck(e.Node.GetName()).GetEndpointMap().Lock()
	defer ss.NIStorage.GetNoCheck(e.Node.GetName()).GetEndpointMap().Unlock()

	if e.Node.GetEndpointMap() == nil {
		return
	}

	ss.NIStorage.GetNoCheck(e.Node.GetName()).GetEndpointMap().Remove(e)
}

func (ss *ServiceInfoStorage) doDownscaling(actualScale, desiredCount int) {
	currentState := ss.Controller.Endpoints
	toEvict := make(map[*core.Endpoint]struct{})

	for i := 0; i < actualScale-desiredCount; i++ {
		endpoint, newState := eviction_policy.EvictionPolicy(currentState)
		if len(currentState) == 0 || endpoint == nil {
			logrus.Errorf("No endpoint to evict in the downscaling loop despite the actual scale is %d.", actualScale)
			panic("End here")
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

	go func() {
		///////////////////////////////////////////////////////////////////////////
		// Firstly, drain the sandboxes and remove them from the data plane(s)
		///////////////////////////////////////////////////////////////////////////
		ss.drainSandbox(toEvict)

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
	}()
}

func (ss *ServiceInfoStorage) drainSandbox(toEvict map[*core.Endpoint]struct{}) {
	////////////////////////////////////////////////////////
	var toDrain []*core.Endpoint
	for elem := range toEvict {
		toDrain = append(toDrain, elem)
	}
	////////////////////////////////////////////////////////
	ss.DataPlaneConnections.Lock()

	wg := sync.WaitGroup{}
	wg.Add(len(ss.DataPlaneConnections.GetMap()))

	for _, dp := range ss.DataPlaneConnections.GetMap() {
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

	ss.DataPlaneConnections.Unlock()
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

func (ss *ServiceInfoStorage) updateEndpoints(endpoints []*proto.EndpointInfo) {
	ss.DataPlaneConnections.Lock()
	defer ss.DataPlaneConnections.Unlock()

	ss.singlethreadUpdateEndpoints(endpoints)
}

func (ss *ServiceInfoStorage) singlethreadUpdateEndpoints(endpoints []*proto.EndpointInfo) {
	wg := &sync.WaitGroup{}
	wg.Add(ss.DataPlaneConnections.Len())

	for _, dp := range ss.DataPlaneConnections.GetMap() {
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

// TODO: Refactor two following function - design can be improved
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

func (ss *ServiceInfoStorage) prepareCurrentEndpointInfoList() []*proto.EndpointInfo {
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
