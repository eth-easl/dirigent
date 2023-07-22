package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane"
	"cluster_manager/internal/control_plane/persistence"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/logger"
	"cluster_manager/pkg/utils"
	"context"
	"github.com/sirupsen/logrus"
	"os/signal"
	"path"
	"syscall"

	"google.golang.org/grpc"
)

func parsePlacementPolicy(controlPlaneConfig config.ControlPlaneConfig) control_plane.PlacementPolicy {
	switch controlPlaneConfig.PlacementPolicy {
	case "random":
		return control_plane.PLACEMENT_RANDOM
	case "round-robin":
		return control_plane.PLACEMENT_ROUND_ROBIN
	case "kubernetes":
		return control_plane.PLACEMENT_KUBERNETES
	default:
		logrus.Error("Failed to parse placement, default policy is random")
		return control_plane.PLACEMENT_RANDOM
	}
}

func main() {
	config, err := config.ReadControlPlaneConfiguration("cmd/master_node/config.yaml")
	if err != nil {
		logrus.Fatal("Failed to read configuration file (error : %s)", err.Error())
	}

	logger.SetupLogger(config.Verbosity)

	redisClient, err := persistence.CreateRedisClient(context.Background(), config.RedisLogin)
	if err != nil {
		logrus.Fatal("Failed to connect to the database (error : %s)", err.Error())
	}

	cpApiServer := api.CreateNewCpApiServer(redisClient, path.Join(config.TraceOutputFolder, "cold_start_trace.csv"), parsePlacementPolicy(config))
	cpApiServer.ReconstructState(context.Background())

	defer cpApiServer.SerializeCpApiServer(context.Background())

	go cpApiServer.CheckPeriodicallyWorkerNodes()

	go cpApiServer.ColdStartTracing.StartTracingService()
	defer close(cpApiServer.ColdStartTracing.InputChannel)

	go api.StartServiceRegistrationServer(cpApiServer, config.PortRegistration)
	go grpc_helpers.CreateGRPCServer(utils.DockerLocalhost, config.Port, func(sr grpc.ServiceRegistrar) {
		proto.RegisterCpiInterfaceServer(sr, cpApiServer)
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		logrus.Info("Received interruption signal, try to gracefully stop")
	}
}
