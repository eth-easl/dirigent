package control_plane

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/autoscaling"
	"cluster_manager/internal/control_plane/core"
	"cluster_manager/pkg/grpc_helpers"
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

/*
Helpers for control_plane.go
*/

func (c *ControlPlane) parseDataplaneRequest(ctx context.Context, in *proto.DataplaneInfo) (proto.DataplaneInformation, error) {
	ipAddress, ok := grpc_helpers.GetIPAddressFromGRPCCall(ctx)
	if !ok {
		logrus.Debug("Failed to extract IP address from data plane registration request")
		return proto.DataplaneInformation{}, errors.New("failed to extract IP address from data plane registration request")
	}

	return proto.DataplaneInformation{
		Address:   ipAddress,
		ApiPort:   strconv.Itoa(int(in.APIPort)),
		ProxyPort: strconv.Itoa(int(in.ProxyPort)),
	}, nil
}

// Only one goroutine will execute the function per service
func (c *ControlPlane) notifyDataplanesAndStartScalingLoop(ctx context.Context, serviceInfo *proto.ServiceInfo, reconstructFromPersistence bool) error {
	c.DataPlaneConnections.RLock()
	for _, conn := range c.DataPlaneConnections.GetMap() {
		_, err := conn.AddDeployment(ctx, serviceInfo)
		if err != nil {
			return err
		}
	}
	c.DataPlaneConnections.RUnlock()

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
		WorkerEndpoints:         c.WorkerEndpoints,
		StartTime:               startTime,
	})

	go c.SIStorage.GetNoCheck(serviceInfo.Name).ScalingControllerLoop(c.NIStorage, c.DataPlaneConnections)

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
