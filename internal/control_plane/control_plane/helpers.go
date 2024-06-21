package control_plane

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/internal/control_plane/control_plane/endpoint_placer"
	"cluster_manager/internal/control_plane/control_plane/endpoint_placer/eviction_policy"
	placement_policy2 "cluster_manager/internal/control_plane/control_plane/endpoint_placer/placement_policy"
	"cluster_manager/internal/control_plane/control_plane/function_state"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/utils"
	"cluster_manager/proto"
	"context"
	"github.com/sirupsen/logrus"
	"sync/atomic"
)

/*
Helpers for control_plane.go
*/

// Only one goroutine will execute the function per service
func (c *ControlPlane) notifyDataplanesAndStartScalingLoop(ctx context.Context, serviceInfo *proto.ServiceInfo) error {

	c.DataPlaneConnections.Lock()
	for _, conn := range c.DataPlaneConnections.GetMap() {
		_, err := conn.AddDeployment(ctx, serviceInfo)
		if err != nil {
			logrus.Warnf("Failed to add deployment to data plane %s - %v", conn.GetIP(), err)
		}
	}
	c.DataPlaneConnections.Unlock()

	functionState := function_state.NewFunctionState(serviceInfo)

	c.autoscalingManager.Create(functionState)

	placementPolicy, evictionPolicy := parsePlacementEvictionPolicies(c.Config)

	c.SIStorage.Set(serviceInfo.Name, &endpoint_placer.EndpointPlacer{
		FunctionState:           functionState,
		ColdStartTracingChannel: c.ColdStartTracing.InputChannel,
		PlacementPolicy:         placementPolicy,
		EvictionPolicy:          evictionPolicy,
		PersistenceLayer:        c.PersistenceLayer,
		NIStorage:               c.NIStorage,
		DataPlaneConnections:    c.DataPlaneConnections,
	})

	go c.SIStorage.GetNoCheck(serviceInfo.Name).ScalingControllerLoop()

	return nil
}

func (c *ControlPlane) removeServiceFromDataplane(ctx context.Context, serviceInfo *proto.ServiceInfo) error {
	c.DataPlaneConnections.Lock()
	for _, conn := range c.DataPlaneConnections.GetMap() {
		_, err := conn.DeleteDeployment(ctx, serviceInfo)
		if err != nil {
			return err
		}
	}
	c.DataPlaneConnections.Unlock()

	return nil
}

func (c *ControlPlane) removeEndpointsAssociatedWithNode(nodeID string) {
	c.SIStorage.Lock()
	for _, value := range c.SIStorage.GetMap() {
		toExclude := make(map[*core.Endpoint]struct{})
		value.FunctionState.EndpointLock.Lock()
		for _, endpoint := range value.FunctionState.Endpoints {
			if endpoint.Node.GetName() == nodeID {
				toExclude[endpoint] = struct{}{}
			}
		}

		value.ExcludeEndpoints(toExclude)
		value.FunctionState.EndpointLock.Unlock()

		atomic.AddInt64(&value.FunctionState.ActualScale, int64(-len(toExclude)))
	}
	c.SIStorage.Unlock()
}

func searchEndpointByContainerName(endpoints []*core.Endpoint, cid string) *core.Endpoint {
	for _, e := range endpoints {
		if e.SandboxID == cid {
			return e
		}
	}

	return nil
}

/*
* Functions used for testing
 */

func (c *ControlPlane) GetNumberConnectedWorkers() int {
	return c.NIStorage.AtomicLen()
}

func (c *ControlPlane) GetNumberDataplanes() int {
	return c.DataPlaneConnections.AtomicLen()
}

func (c *ControlPlane) GetNumberServices() int {
	return c.SIStorage.AtomicLen()
}

// Parse placement and eviction policies

func parsePlacementEvictionPolicies(controlPlaneConfig *config.ControlPlaneConfig) (placement_policy2.PlacementPolicy, eviction_policy.EvictionPolicy) {
	var placementPolicy placement_policy2.PlacementPolicy
	switch controlPlaneConfig.PlacementPolicy {
	case "random":
		placementPolicy = placement_policy2.NewRandomPlacement()
	case "round-robin":
		placementPolicy = placement_policy2.NewRoundRobinPlacement()
	case "kubernetes":
		placementPolicy = placement_policy2.NewKubernetesPolicy()
	default:
		logrus.Error("Failed to parse placement, default policy is random")
		placementPolicy = placement_policy2.NewRandomPlacement()
	}

	return placementPolicy, eviction_policy.NewDefaultevictionPolicy()
}

func isValidAutoscaler(autoscaler string) bool {
	return autoscaler == utils.DEFAULT_AUTOSCALER || autoscaler == utils.PREDICTIVE_AUTOSCALER || autoscaler == utils.MU_AUTOSCALER
}
