package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	"cluster_manager/common"
	"flag"
	"google.golang.org/grpc"
)

var (
	dataPlaneIP      = flag.String("dataPlaneIP", "localhost", "Control plane IP address")
	dataPlanePort    = flag.String("dataPlanePort", common.DefaultDataPlaneApiPort, "Control plane port")
	port             = flag.String("cpPort", common.DefaultControlPlanePort, "Data plane HTTP service registration incoming traffic port")
	portRegistration = flag.String("portRegistration", common.DefaultControlPlanePortServiceRegistration, "Data plane HTTP service registration incoming traffic port")
	verbosity        = flag.String("verbosity", "info", "Logging verbosity - choose from [info, debug, trace]")
)

func main() {
	flag.Parse()
	common.InitLibraries(*verbosity)

	dpiInterface := common.InitializeDataPlaneConnection(*dataPlaneIP, *dataPlanePort)
	cpApiServer := api.CreateNewCpApiServer(dpiInterface)

	go api.StartServiceRegistrationServer(cpApiServer, "0.0.0.0", *portRegistration)
	common.CreateGRPCServer("0.0.0.0", *port, func(sr grpc.ServiceRegistrar) {
		proto.RegisterCpiInterfaceServer(sr, cpApiServer)
	})
}
