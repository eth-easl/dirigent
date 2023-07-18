package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	common "cluster_manager/internal/common"
	"cluster_manager/internal/control_plane"
	config2 "cluster_manager/pkg/config"
	"cluster_manager/pkg/logger"
	"cluster_manager/pkg/utils"
	context2 "context"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"os/signal"
	"path"
	"syscall"

	"google.golang.org/grpc"
)

func parsePlacementPolicy(controlPlaneConfig config2.ControlPlaneConfig) control_plane.PlacementPolicy {
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
	config, err := config2.ReadControlPlaneConfiguration("cmd/master_node/config.yaml")
	if err != nil {
		logrus.Fatal("Failed to read configuration file (error : %s)", err.Error())
	}

	logger.SetupLogger(config.Verbosity)

	redisClient, err := control_plane.CreateRedisClient(context2.Background(), config.RedisLogin)
	if err != nil {
		logrus.Fatal("Failed to connect to the database (error : %s)", err.Error())
	}

	cpApiServer := api.CreateNewCpApiServer(redisClient, path.Join(config.TraceOutputFolder, "cold_start_trace.csv"), parsePlacementPolicy(config))
	cpApiServer.ReconstructState(context2.Background())

	defer cpApiServer.SerializeCpApiServer(context.Background())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go cpApiServer.ColdStartTracing.StartTracingService()
	defer close(cpApiServer.ColdStartTracing.InputChannel)

	go api.StartServiceRegistrationServer(cpApiServer, config.PortRegistration)
	go common.CreateGRPCServer(utils.DockerLocalhost, config.Port, func(sr grpc.ServiceRegistrar) {
		proto.RegisterCpiInterfaceServer(sr, cpApiServer)
	})

	select {
	case <-ctx.Done():
		logrus.Info("Received interruption signal, try to gracefully stop")
	}
}
