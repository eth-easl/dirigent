package api

import (
	"cluster_manager/api/proto"
	"cluster_manager/common"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
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
	ColdStartTracingChannel *chan common.ColdStartLogEntry
}

type Endpoint struct {
	SandboxID string
	URL       string
	Node      *WorkerNode
	HostPort  int32
}

func (ss *ServiceInfoStorage) GetAllURLs() []string {
	var res []string

	for i := 0; i < len(ss.Controller.Endpoints); i++ {
		res = append(res, ss.Controller.Endpoints[i].URL)
	}

	return res
}

func (ss *ServiceInfoStorage) ScalingControllerLoop(nodeList *NodeInfoStorage, dpiClient proto.DpiInterfaceClient) {
	for {
		select {
		case desiredCount := <-*ss.Controller.DesiredStateChannel:

			ss.Controller.Lock()

			actualScale := ss.Controller.ScalingMetadata.ActualScale
			ss.Controller.ScalingMetadata.ActualScale = desiredCount

			if actualScale < desiredCount {
				go ss.doUpscaling(desiredCount-actualScale, nodeList, dpiClient)
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

				go ss.doDownscaling(toEvict, ss.prepareUrlList(), dpiClient)
			}

			ss.Controller.Unlock() // for all cases (>, ==, <)
		}
	}
}

func (ss *ServiceInfoStorage) doUpscaling(toCreateCount int, nodeList *NodeInfoStorage, dpiClient proto.DpiInterfaceClient) {
	barrier := sync.WaitGroup{}

	var finalEndpoint []*Endpoint
	endpointMutex := sync.Mutex{}

	logrus.Debug("Need to create: ", toCreateCount, " sandboxes")

	barrier.Add(toCreateCount)
	for i := 0; i < toCreateCount; i++ {
		go func() {
			defer barrier.Done()

			node := placementPolicy(nodeList)

			ctx, cancel := context.WithTimeout(context.Background(), WorkerNodeTrafficTimeout)
			defer cancel()

			resp, err := node.GetAPI().CreateSandbox(ctx, ss.ServiceInfo)
			*ss.ColdStartTracingChannel <- common.ColdStartLogEntry{
				ServiceName:      ss.ServiceInfo.Name,
				ContainerID:      resp.ID,
				Success:          resp.Success,
				LatencyBreakdown: resp.LatencyBreakdown,
			}

			if err != nil || !resp.Success {
				logrus.Warn("Failed to start a sandbox on worker node ", node.Name)

				ss.Controller.Lock()
				ss.Controller.ScalingMetadata.ActualScale--
				ss.Controller.Unlock()

				return
			}

			logrus.Debug("Sandbox creation took: ", resp.LatencyBreakdown.Total, " ms")

			///////////////////////////////////////////
			endpointMutex.Lock()
			logrus.Debug("Endpoint appended: ", resp.ID)
			finalEndpoint = append(finalEndpoint, &Endpoint{
				SandboxID: resp.ID,
				URL:       fmt.Sprintf("%s:%d", node.IP, resp.PortMappings.HostPort),
				Node:      node,
				HostPort:  resp.PortMappings.HostPort,
			})
			endpointMutex.Unlock()
			///////////////////////////////////////////
		}()
	}

	// batch update of endpoints
	barrier.Wait()

	logrus.Debug("All sandboxes have been created. Updating endpoints.")

	ss.Controller.Lock()
	// no need for 'endpointMutex' as the barrier has already been passed
	ss.Controller.Endpoints = append(ss.Controller.Endpoints, finalEndpoint...)
	urls := ss.prepareUrlList()

	ss.Controller.Unlock()

	logrus.Debug("Propagating endpoints.")
	ss.updateEndpoints(dpiClient, urls)
	logrus.Debug("Endpoints updated.")
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

func (ss *ServiceInfoStorage) doDownscaling(toEvict map[*Endpoint]struct{}, urls []*proto.EndpointInfo, dpiClient proto.DpiInterfaceClient) {
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
		}()
	}

	// batch update of endpoints
	barrier.Wait()
	ss.updateEndpoints(dpiClient, urls)
}

func (ss *ServiceInfoStorage) updateEndpoints(dpiClient proto.DpiInterfaceClient, endpoints []*proto.EndpointInfo) {
	resp, err := dpiClient.UpdateEndpointList(context.Background(), &proto.DeploymentEndpointPatch{
		Service:   ss.ServiceInfo,
		Endpoints: endpoints,
	})
	if err != nil || !resp.Success {
		logrus.Warn("Failed to update endpoint list in the data plane")
	}
}

type WorkerNode struct {
	Name string
	IP   string
	Port string

	CpuUsage    int32
	MemoryUsage int32

	LastHeartbeat time.Time
	api           proto.WorkerNodeInterfaceClient
}

func (w *WorkerNode) GetAPI() proto.WorkerNodeInterfaceClient {
	if w.api == nil {
		w.api = common.InitializeWorkerNodeConnection(w.IP, w.Port)
	}

	return w.api
}
