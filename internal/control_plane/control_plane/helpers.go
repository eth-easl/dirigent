package control_plane

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/internal/control_plane/control_plane/eviction_policy"
	"cluster_manager/internal/control_plane/control_plane/per_function_state"
	"cluster_manager/internal/control_plane/control_plane/placement_policy"
	"cluster_manager/internal/control_plane/control_plane/predictive_autoscaler"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/utils"
	"cluster_manager/proto"
	"context"
	"github.com/sirupsen/logrus"
	"sync/atomic"
	"time"
)

/*
Helpers for control_plane.go
*/

// Only one goroutine will execute the function per service
func (c *ControlPlane) notifyDataplanesAndStartScalingLoop(ctx context.Context, serviceInfo *proto.ServiceInfo, reconstructFromPersistence bool) error {

	c.DataPlaneConnections.Lock()
	for _, conn := range c.DataPlaneConnections.GetMap() {
		_, err := conn.AddDeployment(ctx, serviceInfo)
		if err != nil {
			logrus.Warnf("Failed to add deployment to data plane %s - %v", conn.GetIP(), err)
		}
	}
	c.DataPlaneConnections.Unlock()

	startTime := time.Time{}
	if reconstructFromPersistence {
		startTime = time.Now()
	}

	pfState := per_function_state.NewPerFunctionState(serviceInfo)

	var autoscaler core.AutoscalingInterface

	switch c.Config.Autoscaler {
	case utils.DEFAULT_AUTOSCALER:
		autoscaler = per_function_state.NewDefaultAutoscaler(pfState)
	case utils.PREDICTIVE_AUTOSCALER:
		fallthrough
	case utils.MU_AUTOSCALER:
		// Predictive & mu autoscaling
		c.multiscaler.Create(context.Background(),
			pfState,
			&predictive_autoscaler.Decider{
				Name:                     serviceInfo.Name,
				AutoscalingConfiguration: serviceInfo.AutoscalingConfig,
				Status:                   predictive_autoscaler.DeciderStatus{},
			})
		autoscaler = c.multiscaler
	default:
		logrus.Fatalf("Unknown autoscaler : %s", c.Config.Autoscaler)
	}

	c.SIStorage.Set(serviceInfo.Name, &ServiceInfoStorage{
		Autoscaler:              autoscaler,
		ServiceInfo:             serviceInfo,
		ControlPlane:            c,
		PerFunctionState:        pfState,
		ColdStartTracingChannel: c.ColdStartTracing.InputChannel,
		PlacementPolicy:         c.PlacementPolicy,
		EvictionPolicy:          eviction_policy.NewDefaultevictionPolicy(),
		PersistenceLayer:        c.PersistenceLayer,
		NIStorage:               c.NIStorage,
		DataPlaneConnections:    c.DataPlaneConnections,
		StartTime:               startTime,
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
		value.PerFunctionState.EndpointLock.Lock()
		for _, endpoint := range value.PerFunctionState.Endpoints {
			if endpoint.Node.GetName() == nodeID {
				toExclude[endpoint] = struct{}{}
			}
		}

		value.excludeEndpoints(toExclude)
		value.PerFunctionState.EndpointLock.Unlock()

		atomic.AddInt64(&value.PerFunctionState.ActualScale, int64(-len(toExclude)))
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

// Parse placement policy

func ParsePlacementPolicy(controlPlaneConfig config.ControlPlaneConfig) placement_policy.PlacementPolicy {
	switch controlPlaneConfig.PlacementPolicy {
	case "random":
		return placement_policy.NewRandomPlacement()
	case "round-robin":
		return placement_policy.NewRoundRobinPlacement()
	case "kubernetes":
		return placement_policy.NewKubernetesPolicy()
	default:
		logrus.Error("Failed to parse placement, default policy is random")
		return placement_policy.NewRandomPlacement()
	}
}

func isValidAutoscaler(autoscaler string) bool {
	return autoscaler == utils.DEFAULT_AUTOSCALER || autoscaler == utils.PREDICTIVE_AUTOSCALER || autoscaler == utils.MU_AUTOSCALER
}
