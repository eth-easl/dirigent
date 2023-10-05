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
		Controller:              autoscaling.NewPerFunctionStateController(make(chan int), serviceInfo),
		ColdStartTracingChannel: &c.ColdStartTracing.InputChannel,
		PlacementPolicy:         c.PlacementPolicy,
		PersistenceLayer:        c.PersistenceLayer,
		NodeInformation:         c.NIStorage,
		StartTime:               startTime,
	})

	go c.SIStorage.GetNoCheck(serviceInfo.Name).ScalingControllerLoop(c.NIStorage, c.DataPlaneConnections)

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

	close(c.SIStorage.AtomicGetNoCheck(serviceInfo.Name).Controller.DesiredStateChannel)

	return nil
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
