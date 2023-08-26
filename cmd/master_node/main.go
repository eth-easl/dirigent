package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane"
	"cluster_manager/internal/control_plane/persistence"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/logger"
	"cluster_manager/pkg/profiler"
	"cluster_manager/pkg/utils"
	"context"
	"flag"
	"os/signal"
	"path"
	"syscall"
	"time"

	_ "net/http/pprof"

	"github.com/sirupsen/logrus"

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

	var persistenceLayer persistence.PersistenceLayer

	if config.Persistence {
		persistenceLayer, err = persistence.CreateRedisClient(context.Background(), config.RedisConf)
		if err != nil {
			logrus.Fatalf("Failed to connect to the database (error : %s)", err.Error())
		}
	} else {
		persistenceLayer = persistence.NewEmptyPeristenceLayer()
	}

	cpApiServer := api.CreateNewCpApiServer(persistenceLayer, path.Join(config.TraceOutputFolder, "cold_start_trace.csv"), parsePlacementPolicy(config))

	start := time.Now()

	cpApiServer.ReconstructState(context.Background(), config)

	elapsed := time.Since(start)
	logrus.Infof("Took %s seconds to reconstruct", elapsed)

	defer cpApiServer.SerializeCpApiServer(context.Background())

	go cpApiServer.CheckPeriodicallyWorkerNodes()

	go cpApiServer.ControlPlane.ColdStartTracing.StartTracingService()
	defer close(cpApiServer.ControlPlane.ColdStartTracing.InputChannel)

	go api.StartServiceRegistrationServer(cpApiServer, config.PortRegistration)
	go grpc_helpers.CreateGRPCServer(utils.DockerLocalhost, config.Port, func(sr grpc.ServiceRegistrar) {
		proto.RegisterCpiInterfaceServer(sr, cpApiServer)
	})

	go profiler.SetupProfilerServer(config.Profiler)

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
