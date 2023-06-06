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

	Controller *PFStateController
	endpoints  []Endpoint
}

type Endpoint struct {
	SandboxID string
	URL       string
	Node      *WorkerNode
	HostPort  int32
}

func (ss *ServiceInfoStorage) GetAllURLs() []string {
	var res []string

	for i := 0; i < len(ss.endpoints); i++ {
		res = append(res, ss.endpoints[i].URL)
	}

	return res
}

func (ss *ServiceInfoStorage) ScalingControllerLoop(nodeList *NodeInfoStorage, dpiClient proto.DpiInterfaceClient) {
	// TODO: add locking
	for {
		select {
		case desiredCount := <-*ss.Controller.DesiredStateChannel:
			ss.Controller.Lock()
			actualScale := ss.Controller.ScalingMetadata.ActualScale
			ss.Controller.Unlock()

			if actualScale < desiredCount {
				ss.doUpscaling(actualScale, desiredCount, nodeList, dpiClient)
			} else if actualScale > desiredCount {
				ss.doDownscaling(actualScale, desiredCount, dpiClient)
			}
		}
	}
}

func (ss *ServiceInfoStorage) doUpscaling(actualScale int, desiredCount int, nodeList *NodeInfoStorage, dpiClient proto.DpiInterfaceClient) {
	diff := desiredCount - actualScale
	barrier := sync.WaitGroup{}

	var finalEndpoint []Endpoint
	endpointMutex := sync.Mutex{}

	barrier.Add(diff)
	for i := 0; i < diff; i++ {
		go func() {
			defer barrier.Done()

			node := placementPolicy(nodeList)

			ctx, cancel := context.WithTimeout(context.Background(), WorkerNodeTrafficTimeout)
			defer cancel()

			resp, err := node.GetAPI().CreateSandbox(ctx, ss.ServiceInfo)
			if err != nil || !resp.Success {
				logrus.Warn("Failed to start a sandbox on worker node ", node.Name)
				return
			}

			///////////////////////////////////////////
			endpointMutex.Lock()
			finalEndpoint = append(finalEndpoint, Endpoint{
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

	ss.Controller.Lock()
	ss.endpoints = finalEndpoint // no need for mutex as the barrier has already been passed
	ss.Controller.ScalingMetadata.ActualScale += diff
	ss.Controller.Unlock()

	ss.updateEndpoints(dpiClient)
}

func (ss *ServiceInfoStorage) doDownscaling(actualScale int, desiredCount int, dpiClient proto.DpiInterfaceClient) {
	diff := actualScale - desiredCount
	barrier := sync.WaitGroup{}

	ss.Controller.Lock()
	currentState := ss.endpoints
	var toEvict []*Endpoint
	for i := 0; i < diff; i++ {
		endpoint, newState := evictionPolicy(&currentState)
		if endpoint == nil {
			break
		}

		toEvict = append(toEvict, endpoint)
		currentState = newState
	}
	ss.Controller.Unlock()

	barrier.Add(diff)
	for i := 0; i < diff; i++ {
		index := i

		go func() {
			defer barrier.Done()

			if len(toEvict) == 0 || toEvict[index] == nil || index > len(toEvict) {
				return
			}

			victim := toEvict[index]

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

	ss.Controller.Lock()
	ss.endpoints = currentState
	ss.Controller.ScalingMetadata.ActualScale -= diff
	ss.Controller.Unlock()

	ss.updateEndpoints(dpiClient)
}

func (ss *ServiceInfoStorage) updateEndpoints(dpiClient proto.DpiInterfaceClient) {
	resp, err := dpiClient.UpdateEndpointList(context.Background(), &proto.DeploymentEndpointPatch{
		Service:   ss.ServiceInfo,
		Endpoints: ss.GetAllURLs(),
	})
	if err != nil || !resp.Success {
		logrus.Warn("Failed to update endpoint list in the data plane")
	}
}

type WorkerNode struct {
	Name string
	IP   string
	Port string

	LastHeartbeat time.Time
	api           proto.WorkerNodeInterfaceClient
}

func (w *WorkerNode) GetAPI() proto.WorkerNodeInterfaceClient {
	if w.api == nil {
		w.api = common.InitializeWorkerNodeConnection(w.IP, w.Port)
	}

	return w.api
}
