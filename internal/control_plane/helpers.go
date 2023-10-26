package control_plane

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/autoscaling"
	"cluster_manager/internal/control_plane/core"
	"context"
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
			return err
		}
	}
	c.DataPlaneConnections.Unlock()

	startTime := time.Time{}
	if reconstructFromPersistence {
		startTime = time.Now()
	}

	c.SIStorage.AtomicSet(serviceInfo.Name, &ServiceInfoStorage{
		ServiceInfo:             serviceInfo,
		ControlPlane:            c,
		Controller:              autoscaling.NewPerFunctionStateController(make(chan int), serviceInfo, 2*time.Second),
		ColdStartTracingChannel: &c.ColdStartTracing.InputChannel,
		PlacementPolicy:         c.PlacementPolicy,
		PersistenceLayer:        c.PersistenceLayer,
		NodeInformation:         c.NIStorage,
		StartTime:               startTime,
		ShouldTrace:             c.shouldTrace,
		TracingFile:             c.tracingFile,
	})

	go c.SIStorage.AtomicGetNoCheck(serviceInfo.Name).ScalingControllerLoop(c.NIStorage, c.DataPlaneConnections)

	return nil
}

func (c *ControlPlane) removeServiceFromDataplaneAndStopLoop(ctx context.Context, serviceInfo *proto.ServiceInfo, reconstructFromPersistence bool) error {
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

func (c *ControlPlane) removeEndointsAssociatedWithNode(nodeID string) {
	c.SIStorage.Lock()
	for _, value := range c.SIStorage.GetMap() {
		toExclude := make(map[*core.Endpoint]struct{})
		value.Controller.EndpointLock.Lock()
		for _, endpoint := range value.Controller.Endpoints {
			if endpoint.Node.GetName() == nodeID {
				toExclude[endpoint] = struct{}{}
			}
		}

		value.excludeEndpoints(toExclude)
		value.Controller.EndpointLock.Unlock()
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
	return c.NIStorage.Len()
}

func (c *ControlPlane) GetNumberDataplanes() int {
	return c.DataPlaneConnections.Len()
}

func (c *ControlPlane) GetNumberServices() int {
	return c.SIStorage.Len()
}
