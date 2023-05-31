package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	"cluster_manager/common"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func prepopulate() map[string]*proto.ServiceInfo {
	// TODO: remove in production
	s := make(map[string]*proto.ServiceInfo)

	s["/faas.Executor/Execute"] = &proto.ServiceInfo{
		Name:  "/faas.Executor/Execute",
		Image: "docker.io/cvetkovic/empty_function:latest",
		PortForwarding: []*proto.PortMapping{
			{
				GuestPort: 80,
				Protocol:  proto.L4Protocol_TCP,
			},
		},
	}

	return s
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	cpApiServer := &api.CpApiServer{
		NIStorage: api.NodeInfoStorage{
			NodeInfo: make(map[string]*api.WorkerNode),
		},
		SIStorage: api.ServiceInfoStorage{
			ServiceInfo: prepopulate(), // TODO: remove in production
			Scaling: map[string]*api.Autoscaler{
				"/faas.Executor/Execute": {},
			},
		},
	}
	cpApiServer.DpiInterface = api.InitializeDataPlaneConnection()

	common.CreateGRPCServer(common.ControlPlaneHost, common.ControlPlanePort, func(sr grpc.ServiceRegistrar) {
		proto.RegisterCpiInterfaceServer(sr, cpApiServer)
	})
}
