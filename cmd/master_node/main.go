package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	common "cluster_manager/internal/common"
	config2 "cluster_manager/pkg/config"
	"cluster_manager/pkg/logger"
	"cluster_manager/pkg/utils"
	"path"

	"github.com/sirupsen/logrus"

	"google.golang.org/grpc"
)

func main() {
	config, err := config2.ReadControlPlaneConfiguration("config.yaml")
	if err != nil {
		logrus.Fatal("Failed to read configuration file (error : %s)", err.Error())
	}

	logger.SetupLogger(config.Verbosity)

	cpApiServer := api.CreateNewCpApiServer(path.Join(config.TraceOutputFolder, "cold_start_trace.csv"), config2.ParsePlacementPolicy(config))

	go cpApiServer.ColdStartTracing.StartTracingService()
	defer close(cpApiServer.ColdStartTracing.InputChannel)

	go api.StartServiceRegistrationServer(cpApiServer, config.PortRegistration)
	common.CreateGRPCServer(utils.DockerLocalhost, config.Port, func(sr grpc.ServiceRegistrar) {
		proto.RegisterCpiInterfaceServer(sr, cpApiServer)
	})
}
