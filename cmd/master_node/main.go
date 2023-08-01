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
	"flag"
	"github.com/sirupsen/logrus"
	"os/signal"
	"path"
	"syscall"
	"time"

	"google.golang.org/grpc"
)

var (
	configPath = flag.String("configPath", "cmd/master_node/config.yaml", "Path to the configuration file")
)

func main() {
	flag.Parse()

	logrus.Debugf("Configuration path is : %s", *configPath)

	config, err := config.ReadControlPlaneConfiguration(*configPath)
	if err != nil {
		logrus.Fatalf("Failed to read configuration file (error : %s)", err.Error())
	}

	logger.SetupLogger(config.Verbosity)

	redisClient, err := persistence.CreateRedisClient(context.Background(), config.RedisLogin)
	if err != nil {
		logrus.Fatalf("Failed to connect to the database (error : %s)", err.Error())
	}

	cpApiServer := api.CreateNewCpApiServer(redisClient, path.Join(config.TraceOutputFolder, "cold_start_trace.csv"), parsePlacementPolicy(config))

	start := time.Now()
	cpApiServer.ReconstructState(context.Background())
	elapsed := time.Since(start)
	logrus.Infof("Took %s seconds to reconstruct", elapsed)

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

func parsePlacementPolicy(controlPlaneConfig config.ControlPlaneConfig) control_plane.PlacementPolicy {
	switch controlPlaneConfig.PlacementPolicy {
	case "random":
		return control_plane.NewRandomPolicy()
	case "round-robin":
		return control_plane.NewRoundRobinPolicy()
	case "kubernetes":
		return control_plane.NewKubernetesPolicy()
	default:
		logrus.Error("Failed to parse placement, default policy is random")
		return control_plane.NewRandomPolicy()
	}
}
