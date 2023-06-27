package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	common "cluster_manager/internal/common"
	"flag"
	"path"

	"google.golang.org/grpc"
)

var (
	port              = flag.String("cpPort", common.DefaultControlPlanePort, "Control plane traffic incoming port")
	portRegistration  = flag.String("portRegistration", common.DefaultControlPlanePortServiceRegistration, "HTTP service registration incoming traffic port")
	verbosity         = flag.String("verbosity", "info", "Logging verbosity - choose from [info, debug, trace]")
	traceOutputFolder = flag.String("traceOutputFolder", common.DefaultTraceOutputFolder, "Folder where to write all logs")
)

func main() {
	flag.Parse()
	common.InitLibraries(*verbosity)

	cpApiServer := api.CreateNewCpApiServer(path.Join(*traceOutputFolder, "cold_start_trace.csv"))

	go cpApiServer.ColdStartTracing.StartTracingService()
	defer close(cpApiServer.ColdStartTracing.InputChannel)

	go api.StartServiceRegistrationServer(cpApiServer, *portRegistration)
	common.CreateGRPCServer("0.0.0.0", *port, func(sr grpc.ServiceRegistrar) {
		proto.RegisterCpiInterfaceServer(sr, cpApiServer)
	})
}
