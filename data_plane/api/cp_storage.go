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
	endpoints  []string
}

func (ss *ServiceInfoStorage) ScalingControllerLoop(nodeList *NodeInfoStorage, dpiClient proto.DpiInterfaceClient) {
	// TODO: add locking
	for {
		select {
		case desiredCount := <-*ss.Controller.DesiredStateChannel:
			if ss.Controller.ActualScale < desiredCount {
				diff := desiredCount - ss.Controller.ActualScale

				for i := 0; i < diff; i++ {
					node := placementPolicy(nodeList)
					resp, err := node.GetAPI().CreateSandbox(context.Background(), ss.ServiceInfo)
					if err != nil || !resp.Success {
						logrus.Warn("Failed to start a sandbox on worker node ", node.Name)
					}

					ss.Controller.ActualScale++
					if len(resp.PortMappings) > 1 {
						panic("Not yet implemented")
					}
					nodePort := resp.PortMappings[0].HostPort // TODO: add support for many ports
					ss.endpoints = append(ss.endpoints, fmt.Sprintf("localhost:%d", nodePort))

					if resp.Success {
						resp, err := dpiClient.UpdateEndpointList(context.Background(), &proto.DeploymentEndpointPatch{
							Service:   ss.ServiceInfo,
							Endpoints: ss.endpoints,
						})
						if err != nil || !resp.Success {
							logrus.Warn("Failed to update endpoint list in the data plane")
						}
					}
				}
			} else if ss.Controller.ActualScale > desiredCount {
				// TODO: implement downscaling
			}
		}
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
