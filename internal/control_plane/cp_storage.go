package control_plane

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/common"
	placement2 "cluster_manager/internal/control_plane/placement"
	"cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/tracing"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	WorkerNodeTrafficTimeout = 10 * time.Second
)

type NodeInfoStorage struct {
	sync.Mutex

	NodeInfo map[string]*WorkerNode
}

type ServiceInfoStorage struct {
	ServiceInfo *proto.ServiceInfo

	Controller              *PFStateController
	ColdStartTracingChannel *chan tracing.ColdStartLogEntry

	PlacementPolicy  PlacementPolicy
	PertistenceLayer RedisClient

	WorkerEndpoints     map[string]map[*Endpoint]string
	WorkerEndpointsLock *sync.Mutex
}

type Endpoint struct {
	SandboxID string
	URL       string
	Node      *WorkerNode
	HostPort  int32
}

type WorkerNode struct {
	Name string
	IP   string
	Port string

	CpuUsage    int
	MemoryUsage int

	CpuCores int
	Memory   int

	LastHeartbeat time.Time
	api           proto.WorkerNodeInterfaceClient
}

func (ss *ServiceInfoStorage) GetAllURLs() []string {
	var res []string

	for i := 0; i < len(ss.Controller.Endpoints); i++ {
		res = append(res, ss.Controller.Endpoints[i].URL)
	}

	return res
}

func (ss *ServiceInfoStorage) ScalingControllerLoop(nodeList *NodeInfoStorage, dpiClients map[string]*common.DataPlaneConnectionInfo) {
	for {
		select {
		case desiredCount := <-*ss.Controller.DesiredStateChannel:
			ss.Controller.Lock()

			swapped := false
			var actualScale int
			for !swapped {
				actualScale = int(*ss.Controller.ScalingMetadata.ActualScale)
				swapped = atomic.CompareAndSwapInt64(ss.Controller.ScalingMetadata.ActualScale, int64(actualScale), int64(desiredCount))
			}

			if actualScale < desiredCount {
				go ss.doUpscaling(desiredCount-actualScale, nodeList, dpiClients)
			} else if actualScale > desiredCount {
				currentState := ss.Controller.Endpoints
				toEvict := make(map[*Endpoint]struct{})

				for i := 0; i < actualScale-desiredCount; i++ {
					endpoint, newState := evictionPolicy(currentState)

					if _, ok := toEvict[endpoint]; ok {
						logrus.Warn("Endpoint repetition - this is a bug.")
					}
					toEvict[endpoint] = struct{}{}
					currentState = newState
				}

				if actualScale-desiredCount != len(toEvict) {
					logrus.Warn("downscaling reference error")
				}

				ss.Controller.Endpoints = excludeEndpoints(ss.Controller.Endpoints, toEvict)

				go ss.doDownscaling(toEvict, ss.prepareUrlList(), dpiClients)
			}

			ss.Controller.Unlock() // for all cases (>, ==, <)
		}
	}
}

func (ss *ServiceInfoStorage) doUpscaling(toCreateCount int, nodeList *NodeInfoStorage, dpiClients map[string]*common.DataPlaneConnectionInfo) {
	wg := sync.WaitGroup{}

	var finalEndpoint []*Endpoint

	endpointMutex := sync.Mutex{}

	logrus.Debug("Need to create: ", toCreateCount, " sandboxes")

	wg.Add(toCreateCount)

	for i := 0; i < toCreateCount; i++ {
		go func() {
			defer wg.Done()

			// TODO : @Lazar, We need to ask some resources
			requested := placement2.CreateResourceMap(1, 1)
			node := placementPolicy(ss.PlacementPolicy, nodeList, requested)

			ctx, cancel := context.WithTimeout(context.Background(), WorkerNodeTrafficTimeout)
			defer cancel()

			resp, err := node.GetAPI().CreateSandbox(ctx, ss.ServiceInfo)
			if resp != nil {
				*ss.ColdStartTracingChannel <- tracing.ColdStartLogEntry{
					ServiceName:      ss.ServiceInfo.Name,
					ContainerID:      resp.ID,
					Success:          resp.Success,
					LatencyBreakdown: resp.LatencyBreakdown,
				}
			} else {
				logrus.Errorf("Returned response is nil, can't write to ColdStartTracingChannel")
			}

			if err != nil || !resp.Success {
				logrus.Warn("Failed to start a sandbox on worker node ", node.Name)

				ss.Controller.Lock()
				atomic.AddInt64(ss.Controller.ScalingMetadata.ActualScale, -1)
				ss.Controller.Unlock()

				return
			}

			logrus.Debug("Sandbox creation took: ", resp.LatencyBreakdown.Total.AsDuration().Milliseconds(), " ms")

			// Critical section
			endpointMutex.Lock()
			logrus.Debug("Endpoint appended: ", resp.ID)

			newEndpoint := &Endpoint{
				SandboxID: resp.ID,
				URL:       fmt.Sprintf("%s:%d", node.IP, resp.PortMappings.HostPort),
				Node:      node,
				HostPort:  resp.PortMappings.HostPort,
			}

			// Update worker node structure
			ss.WorkerEndpointsLock.Lock()
			ss.WorkerEndpoints[node.Name][newEndpoint] = ss.ServiceInfo.Name
			ss.WorkerEndpointsLock.Unlock()

			finalEndpoint = append(finalEndpoint, newEndpoint)

			endpointMutex.Unlock()
		}()
	}

	// batch update of endpoints
	wg.Wait()

	logrus.Debug("All sandboxes have been created. Updating endpoints.")

	ss.Controller.Lock()
	// no need for 'endpointMutex' as the barrer has already been passed
	ss.Controller.Endpoints = append(ss.Controller.Endpoints, finalEndpoint...)
	urls := ss.prepareUrlList()

	err := ss.updatePersistenceLayer()
	if err != nil {
		logrus.Fatal("Implement this part")
	}

	ss.Controller.Unlock()

	logrus.Debug("Propagating endpoints.")
	ss.updateEndpoints(dpiClients, urls)
	logrus.Debug("Endpoints updated.")
}

func (ss *ServiceInfoStorage) ReconstructEndpointsFromDatabase(endpoint *Endpoint) {
	ss.Controller.Lock()
	defer ss.Controller.Unlock()

	endpoints := make([]*Endpoint, 0)
	endpoints = append(endpoints, endpoint)

	ss.addEndpoints(endpoints)
}

func (ss *ServiceInfoStorage) addEndpoints(endpoints []*Endpoint) {
	ss.Controller.Endpoints = append(ss.Controller.Endpoints, endpoints...)
}

func (ss *ServiceInfoStorage) updatePersistenceLayer() error {
	endpointsInformations := make([]*proto.Endpoint, 0)
	for _, endpoint := range ss.Controller.Endpoints {
		endpointsInformations = append(endpointsInformations, &proto.Endpoint{
			SandboxID: endpoint.SandboxID,
			URL:       endpoint.URL,
			NodeName:  endpoint.Node.Name,
			HostPort:  endpoint.HostPort,
		})
	}

	return ss.PertistenceLayer.UpdateEndpoints(context.Background(), ss.ServiceInfo.Name, endpointsInformations)
}

func (ss *ServiceInfoStorage) prepareUrlList() []*proto.EndpointInfo {
	var res []*proto.EndpointInfo

	for i := 0; i < len(ss.Controller.Endpoints); i++ {
		res = append(res, &proto.EndpointInfo{
			ID:  ss.Controller.Endpoints[i].SandboxID,
			URL: ss.Controller.Endpoints[i].URL,
		})
	}

	return res
}

func excludeEndpoints(total []*Endpoint, toExclude map[*Endpoint]struct{}) []*Endpoint {
	var result []*Endpoint

	for _, endpoint := range total {
		_, ok := toExclude[endpoint]

		if !ok {
			result = append(result, endpoint)
		}
	}

	return result
}

func excludeSingleEndpoint(total []*Endpoint, toExclude *Endpoint) []*Endpoint {
	var result []*Endpoint

	for _, endpoint := range total {
		if endpoint == toExclude {
			continue
		}
		result = append(result, endpoint)
	}

	return result
}

func (ss *ServiceInfoStorage) doDownscaling(toEvict map[*Endpoint]struct{}, urls []*proto.EndpointInfo, dpiClients map[string]*common.DataPlaneConnectionInfo) {
	barrier := sync.WaitGroup{}

	barrier.Add(len(toEvict))

	for key := range toEvict {
		victim := key

		if victim == nil {
			logrus.Debug("Victim null - should not have happened")
			continue // why this happens?
		}

		go func() {
			defer barrier.Done()

			ctx, cancel := context.WithTimeout(context.Background(), WorkerNodeTrafficTimeout)
			defer cancel()

			resp, err := victim.Node.GetAPI().DeleteSandbox(ctx, &proto.SandboxID{
				ID:       victim.SandboxID,
				HostPort: victim.HostPort,
			})
			if err != nil || !resp.Success {
				logrus.Warn("Failed to delete a sandbox with ID '", victim.SandboxID, "' on worker node '", victim.Node.Name, "'")
				return
			}

			// Update worker node structure
			ss.WorkerEndpointsLock.Lock()
			delete(ss.WorkerEndpoints[key.Node.Name], key)
			ss.WorkerEndpointsLock.Unlock()
		}()
	}

	// batch update of endpoints
	barrier.Wait()

	ss.Controller.Lock()

	err := ss.updatePersistenceLayer()

	if err != nil {
		logrus.Fatal("Implement this part")
	}

	ss.Controller.Unlock()

	ss.updateEndpoints(dpiClients, urls)
}

func (ss *ServiceInfoStorage) RemoveEndpoint(endpointToEvict *Endpoint, dpiClients map[string]*common.DataPlaneConnectionInfo) error {

	ss.Controller.Lock()

	ss.Controller.Endpoints = excludeSingleEndpoint(ss.Controller.Endpoints, endpointToEvict)
	atomic.AddInt64(ss.Controller.ScalingMetadata.ActualScale, -1)

	err := ss.updatePersistenceLayer()
	if err != nil {
		return err
	}

	ss.Controller.Unlock()

	ss.updateEndpoints(dpiClients, ss.prepareUrlList())

	return nil
}

func (ss *ServiceInfoStorage) updateEndpoints(dpiClients map[string]*common.DataPlaneConnectionInfo, endpoints []*proto.EndpointInfo) {
	wg := &sync.WaitGroup{}

	for _, conn := range dpiClients {
		c := conn

		wg.Add(1)

		go func() {
			resp, err := c.Iface.UpdateEndpointList(context.Background(), &proto.DeploymentEndpointPatch{
				Service:   ss.ServiceInfo,
				Endpoints: endpoints,
			})
			if err != nil || !resp.Success {
				logrus.Warn("Failed to update endpoint list in the data plane")
			}

			wg.Done()
		}()
	}

	wg.Wait()
}

func (w *WorkerNode) GetAPI() proto.WorkerNodeInterfaceClient {
	if w.api == nil {
		w.api = grpc_helpers.InitializeWorkerNodeConnection(w.IP, w.Port)
	}

	return w.api
}
