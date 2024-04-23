package control_plane

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/internal/data_plane/haproxy"
	config2 "cluster_manager/pkg/config"
	"cluster_manager/proto"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
	"sync"
	"sync/atomic"
	"time"
)

// ReconstructState Single threaded function - reconstruction happend before starting the control plane
func (c *ControlPlane) ReconstructState(ctx context.Context, config config2.ControlPlaneConfig, haProxyApi *haproxy.API) error {
	if !config.Reconstruct {
		// This is to propagate registration servers to HAProxy
		if haProxyApi != nil {
			haProxyApi.HAProxyReconstructionCallback(c.Config, c.reviseDataplanesInLB)
		}

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

	// Here we can already restart HAProxy as data about the registration
	// servers and data planes have been recovered
	if haProxyApi != nil {
		haProxyApi.HAProxyReconstructionCallback(c.Config, c.reviseDataplanesInLB)
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
		if err := c.reconstructEndpointsState(ctx); err != nil {
			return err
		}
		duration := time.Since(start)
		logrus.Infof("Endpoints reconstruction took : %s", duration)
	}

	return c.PersistenceLayer.SetLeader(ctx)
}

// Single threaded function - reconstruction happened before starting the control plane
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

	logrus.Infof("Reconstructed information for %d data planes", len(dataPlaneValues))
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

		go func() {
			conn := wn.ConnectToWorker()
			if conn != nil {
				c.NIStorage.Lock()
				defer c.NIStorage.Unlock()

				c.NIStorage.Set(wn.GetName(), wn)
			}
		}()
	}

	logrus.Infof("Reconstructed information for %d workers", len(workers))
	return nil
}

// Single threaded function - reconstruction happened before starting the control plane
func (c *ControlPlane) reconstructServiceState(ctx context.Context) error {
	services, err := c.PersistenceLayer.GetServiceInformation(ctx)
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	wg.Add(len(services))

	for _, service := range services {
		go func(service *proto.ServiceInfo) {
			defer wg.Done()

			c.SIStorage.Lock()
			defer c.SIStorage.Unlock()

			err = c.notifyDataplanesAndStartScalingLoop(ctx, service, true)
			if err != nil {
				logrus.Warnf("Failed to reconstruct service state for %s - %v", service.Name, err)
			}
		}(service)
	}

	wg.Wait()

	logrus.Infof("Reconstructed information for %d services", len(services))
	return nil
}

// Single threaded function - reconstruction happens before starting the control plane
func (c *ControlPlane) reconstructEndpointsState(ctx context.Context) error {
	endpoints := make([]*proto.Endpoint, 0)
	for _, workerNode := range c.NIStorage.GetMap() {
		if workerNode == nil {
			logrus.Errorf("Node not found during endpoint listing.")
			continue
		}

		list, err := workerNode.ListEndpoints(ctx, &emptypb.Empty{})
		if err != nil {
			// this probably happens because connection with some of the worker nodes
			// has not been established by the time this line is reached at runtime
			logrus.Errorf("Failed to fetch endpoints from worker node %s - %v", workerNode.GetName(), err)
		}

		endpoints = append(endpoints, list.Endpoint...)
	}

	logrus.Infof("Got information about %d endpoints from worker nodes", len(endpoints))

	for _, e := range endpoints {
		// asynchronous addition of endpoints and propagation to the data plane
		go func(endpoint *proto.Endpoint) {
			c.NIStorage.RLock()
			node, _ := c.NIStorage.Get(endpoint.NodeName)
			if node == nil {
				c.NIStorage.RUnlock()

				logrus.Errorf("Node not found during endpoint reconstruction of service %s.", endpoint.ServiceName)
				return
			}
			c.NIStorage.RUnlock()

			controlPlaneEndpoint := &core.Endpoint{
				SandboxID: endpoint.SandboxID,
				URL:       fmt.Sprintf("%s:%d", node.GetIP(), endpoint.HostPort),
				Node:      node,
				HostPort:  endpoint.HostPort,
			}

			c.SIStorage.RLock()
			ss, found := c.SIStorage.Get(endpoint.ServiceName)
			if !found {
				c.SIStorage.RUnlock()

				logrus.Errorf("Service %s not found during endpoint reconstruction.", endpoint.ServiceName)
				return
			}
			c.SIStorage.RUnlock()

			ss.NIStorage.AtomicGetNoCheck(node.GetName()).GetEndpointMap().AtomicSet(controlPlaneEndpoint, ss.ServiceInfo.Name)

			ss.Controller.EndpointLock.Lock()
			ss.Controller.Endpoints = append(ss.Controller.Endpoints, controlPlaneEndpoint)
			urls := ss.prepareEndpointInfo(ss.Controller.Endpoints)
			ss.Controller.EndpointLock.Unlock()

			ss.updateEndpoints(urls)

			// do not allow downscaling until the system stabilizes
			ss.Controller.ScalingMetadata.InPanicMode = true
			ss.Controller.ScalingMetadata.StartPanickingTimestamp = time.Now()
			atomic.AddInt64(&ss.Controller.ScalingMetadata.ActualScale, 1)

			ss.Controller.Start()
		}(e)
	}

	return nil
}
