package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	"cluster_manager/common"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"sync"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	stopCh := make(chan struct{})

	cpApiServer := &api.CpApiServer{
		NIStorage: api.NodeInfoStorage{
			Mutex:    sync.Mutex{},
			NodeInfo: make(map[string]*api.WorkerNode),
		},
	}
	cpApiServer.DpiInterface = api.InitializeDataPlaneConnection()

	common.CreateGRPCServer(common.ControlPlaneHost, common.ControlPlanePort, func(sr grpc.ServiceRegistrar) {
		proto.RegisterCpiInterfaceServer(sr, cpApiServer)
	})

	<-stopCh
}
