package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	"cluster_manager/common"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	dpiInterface := api.InitializeDataPlaneConnection()
	cpApiServer := api.CreateNewCpApiServer(dpiInterface)

	common.CreateGRPCServer(common.ControlPlaneHost, common.ControlPlanePort, func(sr grpc.ServiceRegistrar) {
		proto.RegisterCpiInterfaceServer(sr, cpApiServer)
	})
}
