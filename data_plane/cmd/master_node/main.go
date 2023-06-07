package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	"cluster_manager/common"
	"flag"
	"google.golang.org/grpc"
)

var (
	port             = flag.String("cpPort", common.DefaultControlPlanePort, "Control plane traffic incoming port")
	portRegistration = flag.String("portRegistration", common.DefaultControlPlanePortServiceRegistration, "HTTP service registration incoming traffic port")
	verbosity        = flag.String("verbosity", "info", "Logging verbosity - choose from [info, debug, trace]")
)

func main() {
	flag.Parse()
	common.InitLibraries(*verbosity)

	cpApiServer := api.CreateNewCpApiServer()

	go api.StartServiceRegistrationServer(cpApiServer, "0.0.0.0", *portRegistration)
	common.CreateGRPCServer("0.0.0.0", *port, func(sr grpc.ServiceRegistrar) {
		proto.RegisterCpiInterfaceServer(sr, cpApiServer)
	})
}
