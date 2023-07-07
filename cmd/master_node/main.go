package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	common "cluster_manager/internal/common"
	"cluster_manager/utils"
	"flag"
	"path"

	"google.golang.org/grpc"
)

var (
	port              = flag.String("cpPort", utils.DefaultControlPlanePort, "Control plane traffic incoming port")
	portRegistration  = flag.String("portRegistration", utils.DefaultControlPlanePortServiceRegistration, "HTTP service registration incoming traffic port")
	verbosity         = flag.String("verbosity", "info", "Logging verbosity - choose from [info, debug, trace]")
	traceOutputFolder = flag.String("traceOutputFolder", utils.DefaultTraceOutputFolder, "Folder where to write all logs")
)

func main() {
	flag.Parse()
	utils.SetupLogger(*verbosity)

	cpApiServer := api.CreateNewCpApiServer(path.Join(*traceOutputFolder, "cold_start_trace.csv"))

	go cpApiServer.ColdStartTracing.StartTracingService()
	defer close(cpApiServer.ColdStartTracing.InputChannel)

	go api.StartServiceRegistrationServer(cpApiServer, *portRegistration)
	common.CreateGRPCServer("0.0.0.0", *port, func(sr grpc.ServiceRegistrar) {
		proto.RegisterCpiInterfaceServer(sr, cpApiServer)
	})
}
