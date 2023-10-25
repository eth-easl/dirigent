package control_plane

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/core"
	config2 "cluster_manager/pkg/config"
	synchronization "cluster_manager/pkg/synchronization"
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
	"sync/atomic"
	"time"
)

// Single threaded function - reconstruction happend before starting the control plane
func (c *ControlPlane) ReconstructState(ctx context.Context, config config2.ControlPlaneConfig) error {
	if !config.Reconstruct {
		return nil
	}

	{
		start := time.Now()
		if err := c.reconstructDataplaneState(ctx); err != nil {
			return err
		}
		duration := time.Since(start)
		logrus.Infof("Data planes reconstruction took : %s", duration)
	}
	{
		start := time.Now()
		if err := c.reconstructWorkersState(ctx); err != nil {
			return err
		}
		duration := time.Since(start)
		logrus.Infof("Worker nodes reconstruction took : %s", duration)
	}
	{
		start := time.Now()
		if err := c.reconstructServiceState(ctx); err != nil {
			return err
		}
		duration := time.Since(start)
		logrus.Infof("Services reconstruction took : %s", duration)
	}
	{
		start := time.Now()
		if err := c.reconstructEndpointsState(ctx, c.DataPlaneConnections); err != nil {
			return err
		}
		duration := time.Since(start)
		logrus.Infof("Endpoints reconstruction took : %s", duration)
	}

	return nil
}

// Single threaded function - reconstruction happend before starting the control plane
func (c *ControlPlane) reconstructDataplaneState(ctx context.Context) error {
	dataPlaneValues, err := c.PersistenceLayer.GetDataPlaneInformation(ctx)
	if err != nil {
		return err
	}

	for _, dataplaneInfo := range dataPlaneValues {
		dataplaneConnection := c.dataPlaneCreator(dataplaneInfo.Address, dataplaneInfo.ApiPort, dataplaneInfo.ProxyPort)
		dataplaneConnection.InitializeDataPlaneConnection(dataplaneInfo.Address, dataplaneInfo.ApiPort)

		c.DataPlaneConnections.Set(dataplaneInfo.Address, dataplaneConnection)
	}

	return nil
}

// Single threaded function - reconstruction happens before starting the control plane
func (c *ControlPlane) reconstructWorkersState(ctx context.Context) error {
	workers, err := c.PersistenceLayer.GetWorkerNodeInformation(ctx)
	if err != nil {
		return err
	}

	for _, worker := range workers {

		wn := c.workerNodeCreator(core.WorkerNodeConfiguration{
			Name:     worker.Name,
			IP:       worker.Ip,
			Port:     worker.Port,
			CpuCores: worker.CpuCores,
			Memory:   worker.Memory,
		})

		c.NIStorage.Set(wn.GetName(), wn)

		go wn.GetAPI()
	}

	return nil
}

// Single threaded function - reconstruction happend before starting the control plane
func (c *ControlPlane) reconstructServiceState(ctx context.Context) error {
	services, err := c.PersistenceLayer.GetServiceInformation(ctx)
	if err != nil {
		return err
	}

	for _, service := range services {
		c.SIStorage.Set(service.Name, &ServiceInfoStorage{})
		err := c.notifyDataplanesAndStartScalingLoop(ctx, service, true)
		if err != nil {
			return err
		}
	}

	return nil
}

// Single threaded function - reconstruction happend before starting the control plane
func (c *ControlPlane) reconstructEndpointsState(ctx context.Context, dpiClients synchronization.SyncStructure[string, core.DataPlaneInterface]) error {
	endpoints := make([]*proto.Endpoint, 0)

	for _, workerNode := range c.NIStorage.GetMap() {
		list, err := workerNode.ListEndpoints(ctx, &emptypb.Empty{})
		if err != nil {
			return err
		}

		endpoints = append(endpoints, list.Endpoint...)
	}

	logrus.Tracef("Found %d endpoints", len(endpoints))

	for _, endpoint := range endpoints {
		node, _ := c.NIStorage.Get(endpoint.NodeName)
		controlPlaneEndpoint := &core.Endpoint{
			SandboxID: endpoint.SandboxID,
			URL:       endpoint.URL,
			Node:      node,
			HostPort:  endpoint.HostPort,
		}

		ss, found := c.SIStorage.Get(endpoint.ServiceName)
		if !found {
			return errors.New("element not found in map")
		}

		ss.Controller.Endpoints = append(ss.Controller.Endpoints, controlPlaneEndpoint)
		atomic.AddInt64(&ss.Controller.ScalingMetadata.ActualScale, 1)

		ss.updateEndpoints(dpiClients, ss.prepareUrlList())
	}

	return nil
}
