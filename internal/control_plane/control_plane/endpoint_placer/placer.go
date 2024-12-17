package endpoint_placer

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/internal/control_plane/control_plane/endpoint_placer/eviction_policy"
	placement_policy2 "cluster_manager/internal/control_plane/control_plane/endpoint_placer/placement_policy"
	"cluster_manager/internal/control_plane/control_plane/image_storage"
	"cluster_manager/internal/control_plane/control_plane/persistence"
	"cluster_manager/internal/control_plane/control_plane/service_state"
	"cluster_manager/pkg/synchronization"
	"cluster_manager/pkg/tracing"
	"cluster_manager/pkg/utils"
	"cluster_manager/proto"
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/durationpb"
)

type EndpointPlacer struct {
	ServiceState            *service_state.ServiceState
	ColdStartTracingChannel chan tracing.ColdStartLogEntry

	PlacementPolicy placement_policy2.PlacementPolicy
	EvictionPolicy  eviction_policy.EvictionPolicy

	PersistenceLayer persistence.PersistenceLayer

	DataPlaneConnections synchronization.SyncStructure[string, core.DataPlaneInterface]
	NIStorage            synchronization.SyncStructure[string, core.WorkerNodeInterface]
	ImageStorage         image_storage.ImageStorage

	DandelionNodes synchronization.SyncStructure[string, bool]
}

func (ep *EndpointPlacer) ScalingControllerLoop() {
	isLoopRunning := true
	desiredCount := 0

	for isLoopRunning {
		desiredCount, isLoopRunning = <-ep.ServiceState.DesiredStateChannel
		loopStarted := time.Now()

		lastValue := false
		for !lastValue {
			select {
			case desiredCount, isLoopRunning = <-ep.ServiceState.DesiredStateChannel:
				lastValue = false
			default:
				lastValue = true
			}
		}

		var actualScale int

		swapped := false
		for !swapped {
			actualScale = int(ep.ServiceState.ActualScale)
			swapped = atomic.CompareAndSwapInt64(&ep.ServiceState.ActualScale, int64(actualScale), int64(desiredCount))
		}

		// Channel closed ==> We send the instruction to remove all endpoints
		if !isLoopRunning {
			desiredCount = 0

			ep.ServiceState.EndpointLock.Lock()
			ep.doDownscaling(actualScale, desiredCount)
			ep.ServiceState.EndpointLock.Unlock()
			break
		}

		ep.ServiceState.EndpointLock.Lock()

		if actualScale < desiredCount {
			ep.doUpscaling(desiredCount-actualScale, loopStarted)
		} else if actualScale > desiredCount {
			ep.doDownscaling(actualScale, desiredCount)
		}

		ep.ServiceState.EndpointLock.Unlock() // for all cases (>, ==, <)
	}
}

func (ep *EndpointPlacer) UpdateDeployment(newServiceInfo *proto.ServiceInfo) {
	ep.DataPlaneConnections.Lock()
	defer ep.DataPlaneConnections.Unlock()

	ep.SingleThreadUpdateDeployment(newServiceInfo)
}

func (ep *EndpointPlacer) SingleThreadUpdateDeployment(newServiceInfo *proto.ServiceInfo) {
	wg := &sync.WaitGroup{}
	wg.Add(ep.DataPlaneConnections.Len())

	for _, dp := range ep.DataPlaneConnections.GetMap() {
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

func (ep *EndpointPlacer) AddEndpoint(endpoint *core.Endpoint) {
	ep.ServiceState.EndpointLock.Lock()
	ep.ServiceState.Endpoints = append(ep.ServiceState.Endpoints, endpoint)
	urls := ep.PrepareEndpointInfo(ep.ServiceState.Endpoints)
	ep.ServiceState.EndpointLock.Unlock()

	ep.UpdateEndpoints(urls)
}

func (ep *EndpointPlacer) RemoveEndpointFromWNStruct(e *core.Endpoint) {
	if e.Node == nil {
		return
	}

	// Update worker node structure
	ep.NIStorage.GetNoCheck(e.Node.GetName()).GetEndpointMap().Lock()
	defer ep.NIStorage.GetNoCheck(e.Node.GetName()).GetEndpointMap().Unlock()

	if e.Node.GetEndpointMap() == nil {
		return
	}

	ep.NIStorage.GetNoCheck(e.Node.GetName()).GetEndpointMap().Remove(e)
}

func (ep *EndpointPlacer) UpdateEndpoints(endpoints []*proto.EndpointInfo) {
	ep.DataPlaneConnections.Lock()
	defer ep.DataPlaneConnections.Unlock()

	ep.SingleThreadUpdateEndpoints(endpoints)
}

func (ep *EndpointPlacer) SingleThreadUpdateEndpoints(endpoints []*proto.EndpointInfo) {
	wg := &sync.WaitGroup{}
	wg.Add(ep.DataPlaneConnections.Len())

	for _, dp := range ep.DataPlaneConnections.GetMap() {
		go func(dataPlane core.DataPlaneInterface) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), utils.WorkerNodeTrafficTimeout)
			defer cancel()

			_, err := dataPlane.UpdateEndpointList(ctx, &proto.DeploymentEndpointPatch{
				ServiceName: ep.ServiceState.ServiceName,
				Endpoints:   endpoints,
			})
			if err != nil {
				logrus.Warnf("Failed to update endpoint list in the data plane - %v", err)
			}
		}(dp)
	}

	wg.Wait()
}

func (ep *EndpointPlacer) ExcludeEndpoints(toExclude map[*core.Endpoint]struct{}) {
	var result []*core.Endpoint

	for _, endpoint := range ep.ServiceState.Endpoints {
		if _, ok := toExclude[endpoint]; !ok {
			result = append(result, endpoint)
		}
	}

	ep.ServiceState.Endpoints = result
}

// PrepareEndpointInfo TODO: Refactor two following function - design can be improved
func (ep *EndpointPlacer) PrepareEndpointInfo(endpoints []*core.Endpoint) []*proto.EndpointInfo {
	var res []*proto.EndpointInfo

	for i := 0; i < len(endpoints); i++ {
		res = append(res, &proto.EndpointInfo{
			ID:  endpoints[i].SandboxID,
			URL: endpoints[i].URL,
		})
	}

	return res
}

func (ep *EndpointPlacer) PrepareCurrentEndpointInfoList() []*proto.EndpointInfo {
	var res []*proto.EndpointInfo

	for i := 0; i < len(ep.ServiceState.Endpoints); i++ {
		res = append(res, &proto.EndpointInfo{
			ID:  ep.ServiceState.Endpoints[i].SandboxID,
			URL: ep.ServiceState.Endpoints[i].URL,
		})
	}

	return res
}

func (ep *EndpointPlacer) doUpscaling(toCreateCount int, loopStarted time.Time) {
	wg := sync.WaitGroup{}

	logrus.Debug("Need to create: ", toCreateCount, " sandboxes")

	wg.Add(toCreateCount)

	for i := 0; i < toCreateCount; i++ {
		go func() {
			defer wg.Done()

			requested := placement_policy2.CreateResourceMap(ep.ServiceState.GetRequestedCpu(), ep.ServiceState.GetRequestedMemory(), ep.ServiceState.FunctionInfo[0].Image)
			node := placement_policy2.ApplyPlacementPolicy(ep.PlacementPolicy, ep.NIStorage, ep.ImageStorage, requested, &ep.DandelionNodes)
			if node == nil {
				logrus.Warn("Failed to do placement. No nodes are schedulable.")

				atomic.AddInt64(&ep.ServiceState.ActualScale, -1)
				return
			}
			node.AddUsage(ep.ServiceState.FunctionInfo[0].GetRequestedCpu(), ep.ServiceState.FunctionInfo[0].GetRequestedMemory())

			ctx, cancel := context.WithTimeout(context.Background(), utils.WorkerNodeTrafficTimeout)
			defer cancel()

			var err error
			var resp *proto.SandboxCreationStatus
			if ep.ServiceState.TaskInfo == nil {
				resp, err = node.CreateSandbox(ctx, ep.ServiceState.FunctionInfo[0])
			} else {
				for _, fInfo := range ep.ServiceState.FunctionInfo { // make sure all task functions are registered in runtime
					_, err = node.CreateSandbox(ctx, fInfo)
					if err != nil {
						logrus.Errorf("Failed to do create sandbox for task function '%s': %v", fInfo.Name, err)
						break
					}
				}
				resp, err = node.CreateTaskSandbox(ctx, ep.ServiceState.TaskInfo)
			}
			if err != nil || !resp.Success {
				text := ""
				if err != nil {
					text += err.Error()
				}
				logrus.Warnf("Failed to start a sandbox on worker node %s (error %s)", node.GetName(), text)

				atomic.AddInt64(&ep.ServiceState.ActualScale, -1)
				node.AddUsage(-ep.ServiceState.FunctionInfo[0].GetRequestedCpu(), -ep.ServiceState.FunctionInfo[0].GetRequestedMemory())

				var latencyBreakdown *proto.SandboxCreationBreakdown
				if resp != nil {
					latencyBreakdown = resp.LatencyBreakdown
				} else {
					latencyBreakdown = &proto.SandboxCreationBreakdown{
						Total: durationpb.New(utils.WorkerNodeTrafficTimeout),
					}
				}
				ep.ColdStartTracingChannel <- tracing.ColdStartLogEntry{
					ServiceName:      ep.ServiceState.ServiceName,
					ContainerID:      "",
					Success:          false,
					PersistenceCost:  0,
					LatencyBreakdown: latencyBreakdown,
				}
				return
			}
			ep.ImageStorage.Lock()
			ep.ImageStorage.RegisterNoFetch(ep.ServiceState.FunctionInfo[0].Image, 0, node)
			ep.ImageStorage.Unlock()

			sandboxCreationTook := time.Since(loopStarted)
			resp.LatencyBreakdown.Total = durationpb.New(sandboxCreationTook)
			logrus.Debug("Sandbox creation took: ", sandboxCreationTook.Milliseconds(), " ms")

			costPersistStart := time.Now()

			newEndpoint := &core.Endpoint{
				SandboxID: resp.ID,
				URL:       resp.URL,
				Node:      node,
				CreationHistory: tracing.ColdStartLogEntry{
					ServiceName:      ep.ServiceState.ServiceName,
					ContainerID:      resp.ID,
					Success:          resp.Success,
					PersistenceCost:  time.Since(costPersistStart),
					LatencyBreakdown: resp.LatencyBreakdown,
				},
			}

			startEndpointPropagation := time.Now()

			// Update worker node structure
			ep.NIStorage.AtomicGetNoCheck(node.GetName()).GetEndpointMap().AtomicSet(newEndpoint, ep.ServiceState.ServiceName)

			ep.ServiceState.Endpoints = append(ep.ServiceState.Endpoints, newEndpoint)
			urls := ep.PrepareEndpointInfo([]*core.Endpoint{newEndpoint}) // prepare delta for sending

			ep.UpdateEndpoints(urls)
			logrus.Debugf("Endpoint has been propagated - %s", resp.ID)

			newEndpoint.CreationHistory.LatencyBreakdown.DataplanePropagation = durationpb.New(time.Since(startEndpointPropagation))
			newEndpoint.CreationHistory.Event = "CREATE"

			ep.ColdStartTracingChannel <- newEndpoint.CreationHistory
		}()
	}

	wg.Wait()
}

func (ep *EndpointPlacer) doDownscaling(actualScale, desiredCount int) {
	currentState := ep.ServiceState.Endpoints
	toEvict := make(map[*core.Endpoint]struct{})

	for i := 0; i < actualScale-desiredCount; i++ {
		endpoint, newState := ep.EvictionPolicy.Evict(currentState)
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

	ep.ExcludeEndpoints(toEvict)

	go func() {
		///////////////////////////////////////////////////////////////////////////
		// Firstly, drain the sandboxes and remove them from the data plane(s)
		///////////////////////////////////////////////////////////////////////////
		ep.drainSandbox(toEvict)

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

				ep.ColdStartTracingChannel <- tracing.ColdStartLogEntry{
					ServiceName:      ep.ServiceState.ServiceName,
					ContainerID:      victim.SandboxID,
					Event:            "DELETE",
					Success:          true,
					PersistenceCost:  0,
					LatencyBreakdown: &proto.SandboxCreationBreakdown{},
				}

				ep.deleteSandbox(victim)
				ep.DandelionNodes.AtomicRemove(victim.Node.GetName())
				ep.RemoveEndpointFromWNStruct(victim)
			}(key)
		}

		// batch update of endpoints
		wg.Wait()
	}()
}

func (ep *EndpointPlacer) deleteSandbox(key *core.Endpoint) {
	if key.Node == nil {
		logrus.Warnf("Reference to a node on sandbox deletion not found. Ignoring request.")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), utils.WorkerNodeTrafficTimeout)
	defer cancel()

	_, err := key.Node.DeleteSandbox(ctx, &proto.SandboxID{ID: key.SandboxID})
	if err != nil {
		logrus.Warnf("Failed to delete a sandbox with ID %s on worker node %s. (error : %v)", key.SandboxID, key.Node.GetName(), err)
	}
}

func (ep *EndpointPlacer) drainSandbox(toEvict map[*core.Endpoint]struct{}) {
	////////////////////////////////////////////////////////
	var toDrain []*core.Endpoint
	for elem := range toEvict {
		toDrain = append(toDrain, elem)
	}
	////////////////////////////////////////////////////////
	ep.DataPlaneConnections.Lock()

	wg := sync.WaitGroup{}
	wg.Add(len(ep.DataPlaneConnections.GetMap()))

	for _, dp := range ep.DataPlaneConnections.GetMap() {
		go func(dataPlane core.DataPlaneInterface) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), utils.WorkerNodeTrafficTimeout)
			defer cancel()

			_, err := dataPlane.DrainSandbox(ctx, &proto.DeploymentEndpointPatch{
				ServiceName: ep.ServiceState.ServiceName,
				Endpoints:   ep.PrepareEndpointInfo(toDrain),
			})
			if err != nil {
				logrus.Errorf("Error draining endpoints for service %s.", ep.ServiceState.ServiceName)
			}
		}(dp)
	}

	ep.DataPlaneConnections.Unlock()
	wg.Wait()
}
