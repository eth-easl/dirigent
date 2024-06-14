package endpoint_placer

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/internal/control_plane/control_plane/endpoint_placer/eviction_policy"
	placement_policy2 "cluster_manager/internal/control_plane/control_plane/endpoint_placer/placement_policy"
	"cluster_manager/internal/control_plane/control_plane/function_state"
	"cluster_manager/internal/control_plane/control_plane/persistence"
	"cluster_manager/pkg/synchronization"
	"cluster_manager/pkg/tracing"
	"cluster_manager/pkg/utils"
	"cluster_manager/proto"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/durationpb"
)

type EndpointPlacer struct {
	FunctionState           *function_state.FunctionState
	ColdStartTracingChannel chan tracing.ColdStartLogEntry

	PlacementPolicy placement_policy2.PlacementPolicy
	EvictionPolicy  eviction_policy.EvictionPolicy

	PersistenceLayer persistence.PersistenceLayer

	DataPlaneConnections synchronization.SyncStructure[string, core.DataPlaneInterface]
	NIStorage            synchronization.SyncStructure[string, core.WorkerNodeInterface]

	DandelionNodes synchronization.SyncStructure[string, bool]
}

func (ss *EndpointPlacer) ScalingControllerLoop() {
	isLoopRunning := true
	desiredCount := 0

	for isLoopRunning {
		desiredCount, isLoopRunning = <-ss.FunctionState.DesiredStateChannel
		loopStarted := time.Now()

		lastValue := false
		for !lastValue {
			select {
			case desiredCount, isLoopRunning = <-ss.FunctionState.DesiredStateChannel:
				lastValue = false
			default:
				lastValue = true
			}
		}

		var actualScale int

		swapped := false
		for !swapped {
			actualScale = int(ss.FunctionState.ActualScale)
			swapped = atomic.CompareAndSwapInt64(&ss.FunctionState.ActualScale, int64(actualScale), int64(desiredCount))
		}

		// Channel closed ==> We send the instruction to remove all endpoints
		if !isLoopRunning {
			desiredCount = 0

			ss.FunctionState.EndpointLock.Lock()
			ss.doDownscaling(actualScale, desiredCount)
			ss.FunctionState.EndpointLock.Unlock()
			break
		}

		ss.FunctionState.EndpointLock.Lock()

		if actualScale < desiredCount {
			ss.doUpscaling(desiredCount-actualScale, loopStarted)
		} else if actualScale > desiredCount {
			ss.doDownscaling(actualScale, desiredCount)
		}

		ss.FunctionState.EndpointLock.Unlock() // for all cases (>, ==, <)
	}
}

func (ss *EndpointPlacer) UpdateDeployment(newServiceInfo *proto.ServiceInfo) {
	ss.DataPlaneConnections.Lock()
	defer ss.DataPlaneConnections.Unlock()

	ss.SingleThreadUpdateDeployment(newServiceInfo)
}

func (ss *EndpointPlacer) SingleThreadUpdateDeployment(newServiceInfo *proto.ServiceInfo) {
	wg := &sync.WaitGroup{}
	wg.Add(ss.DataPlaneConnections.Len())

	for _, dp := range ss.DataPlaneConnections.GetMap() {
		go func(dataPlane core.DataPlaneInterface) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), utils.WorkerNodeTrafficTimeout)
			defer cancel()

			_, err := dataPlane.UpdateDeployment(ctx, newServiceInfo)
			if err != nil {
				logrus.Warnf("Failed to update deployment in the data plane - %v", err)
			}
		}(dp)
	}

	wg.Wait()
}

func (ss *EndpointPlacer) AddEndpoint(endpoint *core.Endpoint) {
	ss.FunctionState.EndpointLock.Lock()
	ss.FunctionState.Endpoints = append(ss.FunctionState.Endpoints, endpoint)
	urls := ss.PrepareEndpointInfo(ss.FunctionState.Endpoints)
	ss.FunctionState.EndpointLock.Unlock()

	ss.UpdateEndpoints(urls)
}

func (ss *EndpointPlacer) RemoveEndpointFromWNStruct(e *core.Endpoint) {
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

func (ss *EndpointPlacer) UpdateEndpoints(endpoints []*proto.EndpointInfo) {
	ss.DataPlaneConnections.Lock()
	defer ss.DataPlaneConnections.Unlock()

	ss.SingleThreadUpdateEndpoints(endpoints)
}

func (ss *EndpointPlacer) SingleThreadUpdateEndpoints(endpoints []*proto.EndpointInfo) {
	wg := &sync.WaitGroup{}
	wg.Add(ss.DataPlaneConnections.Len())

	for _, dp := range ss.DataPlaneConnections.GetMap() {
		go func(dataPlane core.DataPlaneInterface) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), utils.WorkerNodeTrafficTimeout)
			defer cancel()

			_, err := dataPlane.UpdateEndpointList(ctx, &proto.DeploymentEndpointPatch{
				Service:   ss.FunctionState.ServiceInfo,
				Endpoints: endpoints,
			})
			if err != nil {
				logrus.Warnf("Failed to update endpoint list in the data plane - %v", err)
			}
		}(dp)
	}

	wg.Wait()
}

func (ss *EndpointPlacer) ExcludeEndpoints(toExclude map[*core.Endpoint]struct{}) {
	var result []*core.Endpoint

	for _, endpoint := range ss.FunctionState.Endpoints {
		if _, ok := toExclude[endpoint]; !ok {
			result = append(result, endpoint)
		}
	}

	ss.FunctionState.Endpoints = result
}

// TODO: Refactor two following function - design can be improved
func (ss *EndpointPlacer) PrepareEndpointInfo(endpoints []*core.Endpoint) []*proto.EndpointInfo {
	var res []*proto.EndpointInfo

	for i := 0; i < len(endpoints); i++ {
		res = append(res, &proto.EndpointInfo{
			ID:  endpoints[i].SandboxID,
			URL: endpoints[i].URL,
		})
	}

	return res
}

func (ss *EndpointPlacer) PrepareCurrentEndpointInfoList() []*proto.EndpointInfo {
	var res []*proto.EndpointInfo

	for i := 0; i < len(ss.FunctionState.Endpoints); i++ {
		res = append(res, &proto.EndpointInfo{
			ID:  ss.FunctionState.Endpoints[i].SandboxID,
			URL: ss.FunctionState.Endpoints[i].URL,
		})
	}

	return res
}

func (ss *EndpointPlacer) doUpscaling(toCreateCount int, loopStarted time.Time) {
	wg := sync.WaitGroup{}

	logrus.Debug("Need to create: ", toCreateCount, " sandboxes")

	wg.Add(toCreateCount)

	for i := 0; i < toCreateCount; i++ {
		go func() {
			defer wg.Done()

			requested := placement_policy2.CreateResourceMap(ss.FunctionState.ServiceInfo.GetRequestedCpu(), ss.FunctionState.ServiceInfo.GetRequestedMemory())
			node := placement_policy2.ApplyPlacementPolicy(ss.PlacementPolicy, ss.NIStorage, requested, &ss.DandelionNodes)
			if node == nil {
				logrus.Warn("Failed to do placement. No nodes are schedulable.")

				atomic.AddInt64(&ss.FunctionState.ActualScale, -1)
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), utils.WorkerNodeTrafficTimeout)
			defer cancel()

			resp, err := node.CreateSandbox(ctx, ss.FunctionState.ServiceInfo)
			if err != nil || !resp.Success {
				text := ""
				if err != nil {
					text += err.Error()
				}
				logrus.Warnf("Failed to start a sandbox on worker node %s (error %s)", node.GetName(), text)

				atomic.AddInt64(&ss.FunctionState.ActualScale, -1)

				var latencyBreakdown *proto.SandboxCreationBreakdown
				if resp != nil {
					latencyBreakdown = resp.LatencyBreakdown
				} else {
					latencyBreakdown = &proto.SandboxCreationBreakdown{
						Total: durationpb.New(utils.WorkerNodeTrafficTimeout),
					}
				}
				ss.ColdStartTracingChannel <- tracing.ColdStartLogEntry{
					ServiceName:      ss.FunctionState.ServiceInfo.Name,
					ContainerID:      "",
					Success:          false,
					PersistenceCost:  0,
					LatencyBreakdown: latencyBreakdown,
				}
				return
			}

			sandboxCreationTook := time.Since(loopStarted)
			resp.LatencyBreakdown.Total = durationpb.New(sandboxCreationTook)
			logrus.Debug("Sandbox creation took: ", sandboxCreationTook.Milliseconds(), " ms")

			costPersistStart := time.Now()

			newEndpoint := &core.Endpoint{
				SandboxID: resp.ID,
				URL:       fmt.Sprintf("%s:%d", node.GetIP(), resp.PortMappings.HostPort),
				Node:      node,
				HostPort:  resp.PortMappings.HostPort,
				CreationHistory: tracing.ColdStartLogEntry{
					ServiceName:      ss.FunctionState.ServiceInfo.Name,
					ContainerID:      resp.ID,
					Success:          resp.Success,
					PersistenceCost:  time.Since(costPersistStart),
					LatencyBreakdown: resp.LatencyBreakdown,
				},
			}

			startEndpointPropagation := time.Now()

			// Update worker node structure
			ss.NIStorage.AtomicGetNoCheck(node.GetName()).GetEndpointMap().AtomicSet(newEndpoint, ss.FunctionState.ServiceInfo.Name)

			ss.FunctionState.Endpoints = append(ss.FunctionState.Endpoints, newEndpoint)
			urls := ss.PrepareEndpointInfo([]*core.Endpoint{newEndpoint}) // prepare delta for sending

			ss.UpdateEndpoints(urls)
			logrus.Debugf("Endpoint has been propagated - %s", resp.ID)

			newEndpoint.CreationHistory.LatencyBreakdown.DataplanePropagation = durationpb.New(time.Since(startEndpointPropagation))

			ss.ColdStartTracingChannel <- newEndpoint.CreationHistory
		}()
	}

	wg.Wait()
}

func (ss *EndpointPlacer) doDownscaling(actualScale, desiredCount int) {
	currentState := ss.FunctionState.Endpoints
	toEvict := make(map[*core.Endpoint]struct{})

	for i := 0; i < actualScale-desiredCount; i++ {
		endpoint, newState := ss.EvictionPolicy.Evict(currentState)
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

	ss.ExcludeEndpoints(toEvict)

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

				ss.deleteSandbox(victim)
				ss.DandelionNodes.AtomicRemove(victim.Node.GetName())
				ss.RemoveEndpointFromWNStruct(victim)
			}(key)
		}

		// batch update of endpoints
		wg.Wait()
	}()
}

func (ss *EndpointPlacer) deleteSandbox(key *core.Endpoint) {
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

func (ss *EndpointPlacer) drainSandbox(toEvict map[*core.Endpoint]struct{}) {
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
				Service:   ss.FunctionState.ServiceInfo,
				Endpoints: ss.PrepareEndpointInfo(toDrain),
			})
			if err != nil {
				logrus.Errorf("Error draining endpoints for service %s.", ss.FunctionState.ServiceInfo.Name)
			}
		}(dp)
	}

	ss.DataPlaneConnections.Unlock()
	wg.Wait()
}
