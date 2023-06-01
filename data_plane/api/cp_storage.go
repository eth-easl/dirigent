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

type NodeInfoStorage struct {
	sync.Mutex

	NodeInfo map[string]*WorkerNode
}

type ServiceInfoStorage struct {
	ServiceInfo *proto.ServiceInfo
	Scaling     *Autoscaler

	Controller *PFStateController
	endpoints  []Endpoint
}

type Endpoint struct {
	SandboxID string
	URL       string
	Node      *WorkerNode
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
			if ss.Controller.ActualScale < desiredCount {
				ss.doUpscaling(desiredCount, nodeList, dpiClient)
			} else if ss.Controller.ActualScale > desiredCount {
				ss.doDownscaling(desiredCount, dpiClient)
			}
		}
	}
}

func (ss *ServiceInfoStorage) doUpscaling(desiredCount int, nodeList *NodeInfoStorage, dpiClient proto.DpiInterfaceClient) {
	diff := desiredCount - ss.Controller.ActualScale

	for i := 0; i < diff; i++ {
		node := placementPolicy(nodeList)
		resp, err := node.GetAPI().CreateSandbox(context.Background(), ss.ServiceInfo)
		if err != nil || !resp.Success {
			logrus.Warn("Failed to start a sandbox on worker node ", node.Name)
			continue
		}

		ss.Controller.ActualScale++
		ss.endpoints = append(ss.endpoints, Endpoint{
			SandboxID: resp.ID,
			URL:       fmt.Sprintf("localhost:%d", resp.PortMappings.HostPort),
			Node:      node,
		})
		ss.updateEndpoints(dpiClient)
	}
}

func (ss *ServiceInfoStorage) doDownscaling(desiredCount int, dpiClient proto.DpiInterfaceClient) {
	diff := ss.Controller.ActualScale - desiredCount

	for i := 0; i < diff; i++ {
		toEvict, newEndpoint := evictionPolicy(&ss.endpoints)
		resp, err := toEvict.Node.GetAPI().DeleteSandbox(context.Background(), &proto.SandboxID{ID: toEvict.SandboxID})
		if err != nil || !resp.Success {
			logrus.Warn("Failed to delete a sandbox with ID '", toEvict.SandboxID, "' on worker node '", toEvict.Node.Name, "'")
			continue
		}

		ss.Controller.ActualScale--
		ss.endpoints = newEndpoint
		ss.updateEndpoints(dpiClient)
	}
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
		// TODO: remove hardcoded IP address
		w.api = common.InitializeWorkerNodeConnection("localhost", w.Port)
	}

	return w.api
}
