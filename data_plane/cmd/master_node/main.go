package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	"cluster_manager/common"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"time"
)

func prepopulate() *proto.ServiceInfo {
	return &proto.ServiceInfo{
		Name:  "/faas.Executor/Execute",
		Image: "docker.io/cvetkovic/empty_function:latest",
		PortForwarding: []*proto.PortMapping{
			{
				GuestPort: 80,
				Protocol:  proto.L4Protocol_TCP,
			},
		},
	}
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	cpApiServer := &api.CpApiServer{
		NIStorage: api.NodeInfoStorage{
			NodeInfo: make(map[string]*api.WorkerNode),
		},
		SIStorage: make(map[string]*api.ServiceInfoStorage),
	}

	scalingCountCh := make(chan int)
	testService := &api.ServiceInfoStorage{
		ServiceInfo: prepopulate(), // TODO: remove in production
		Scaling: &api.Autoscaler{
			Period:        2 * time.Second,
			NotifyChannel: &scalingCountCh,
		},
		Controller: &api.PFStateController{
			DesiredStateChannel: &scalingCountCh,
		},
	}
	cpApiServer.SIStorage["/faas.Executor/Execute"] = testService

	cpApiServer.DpiInterface = api.InitializeDataPlaneConnection()
	go testService.ScalingControllerLoop(&cpApiServer.NIStorage, cpApiServer.DpiInterface)

	common.CreateGRPCServer(common.ControlPlaneHost, common.ControlPlanePort, func(sr grpc.ServiceRegistrar) {
		proto.RegisterCpiInterfaceServer(sr, cpApiServer)
	})
}
