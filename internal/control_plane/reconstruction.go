package control_plane

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/core"
	config2 "cluster_manager/pkg/config"
	synchronization "cluster_manager/pkg/synchronization"
	"cluster_manager/pkg/utils"
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
	"sync"
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

		go wn.ConnectToWorker()
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

// Single threaded function - reconstruction happens before starting the control plane
func (c *ControlPlane) reconstructEndpointsState(ctx context.Context, dpiClients synchronization.SyncStructure[string, core.DataPlaneInterface]) error {
	endpoints := make([]*proto.Endpoint, 0)
	wg := sync.WaitGroup{}

	dpiClients.Lock()
	for _, dp := range dpiClients.GetMap() {
		wg.Add(1)

		go func(dataPlane core.DataPlaneInterface) {
			defer wg.Done()

			ctxOp, cancel := context.WithTimeout(ctx, utils.WorkerNodeTrafficTimeout)
			defer cancel()

			_, err := dataPlane.UpdateEndpointList(ctxOp, &proto.DeploymentEndpointPatch{
				Service:   nil,
				Endpoints: nil,
			})
			if err != nil {
				logrus.Warnf("Failed to remove all endpoints from the data plane %s - %v", dataPlane.GetIP(), err)
			}
		}(dp)
	}
	dpiClients.Unlock()
	wg.Wait()
	logrus.Debugf("Removed all endpoints from all data planes.")

	for _, workerNode := range c.NIStorage.GetMap() {
		list, err := workerNode.ListEndpoints(ctx, &emptypb.Empty{})
		if err != nil {
			return err
		}

		endpoints = append(endpoints, list.Endpoint...)
	}

	logrus.Debugf("Worker nodes reported %d endpoints during reconstruction", len(endpoints))

	for _, endpoint := range endpoints {
		node, _ := c.NIStorage.Get(endpoint.NodeName)
		if node == nil {
			logrus.Errorf("Node not found during endpoint reconstruction.")
			continue
		}

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

		ss.NodeInformation.AtomicGetNoCheck(node.GetName()).GetEndpointMap().AtomicSet(controlPlaneEndpoint, ss.ServiceInfo.Name)

		ss.Controller.EndpointLock.Lock()
		ss.Controller.Endpoints = append(ss.Controller.Endpoints, controlPlaneEndpoint)
		urls := ss.prepareEndpointInfo(ss.Controller.Endpoints)
		ss.Controller.EndpointLock.Unlock()

		ss.updateEndpoints(dpiClients, urls)

		// do not allow downscaling until the system stabilizes
		ss.Controller.ScalingMetadata.InPanicMode = true
		ss.Controller.ScalingMetadata.StartPanickingTimestamp = time.Now()
		atomic.AddInt64(&ss.Controller.ScalingMetadata.ActualScale, 1)

		ss.Controller.Start()
	}

	return nil
}
