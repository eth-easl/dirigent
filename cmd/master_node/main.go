package main

import (
	"cluster_manager/api"
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/data_plane"
	"cluster_manager/internal/control_plane/data_plane/empty_dataplane"
	"cluster_manager/internal/control_plane/persistence"
	"cluster_manager/internal/control_plane/placement_policy"
	"cluster_manager/internal/control_plane/registration_server"
	"cluster_manager/internal/control_plane/workers"
	"cluster_manager/internal/control_plane/workers/empty_worker"
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
	configPath = flag.String("config", "cmd/master_node/config.yaml", "Path to the configuration file")
)

func main() {
	flag.Parse()
	logrus.Debugf("Configuration path is : %s", *configPath)

	cfg, err := config.ReadControlPlaneConfiguration(*configPath)
	if err != nil {
		logrus.Fatalf("Failed to read configuration file (error : %s)", err.Error())
	}

	logger.SetupLogger(cfg.Verbosity)

	var persistenceLayer persistence.PersistenceLayer

	if cfg.Persistence {
		persistenceLayer, err = persistence.CreateRedisClient(context.Background(), cfg.RedisConf)
		if err != nil {
			logrus.Fatalf("Failed to connect to the database (error : %s)", err.Error())
		}
	} else {
		persistenceLayer = persistence.NewEmptyPeristenceLayer()
	}

	dataplaneCreator := data_plane.NewDataplaneConnection
	workerNodeCreator := workers.NewWorkerNode

	if cfg.RemoveWorkerNode {
		workerNodeCreator = empty_worker.NewEmptyWorkerNode
	}

	if cfg.RemoveDataplane {
		dataplaneCreator = empty_dataplane.NewDataplaneConnectionEmpty
	}

	if cfg.PrecreateSnapshots {
		logrus.Warn("Firecracker snapshot precreation is enabled. Make sure snapshots are enabled on each worker node.")
	}

	cpApiServer := api.CreateNewCpApiServer(persistenceLayer, path.Join(cfg.TraceOutputFolder, "cold_start_trace.csv"), parsePlacementPolicy(cfg), dataplaneCreator, workerNodeCreator, &cfg)

	start := time.Now()

	err = cpApiServer.ReconstructState(context.Background(), cfg)
	if err != nil {
		logrus.Fatalf("Failed to reconstruct state (error : %s)", err.Error())
	}

	elapsed := time.Since(start)
	logrus.Infof("Took %s to reconstruct", elapsed)

	go cpApiServer.CheckPeriodicallyWorkerNodes()

	go cpApiServer.ControlPlane.ColdStartTracing.StartTracingService()
	defer close(cpApiServer.ControlPlane.ColdStartTracing.InputChannel)

	go registration_server.StartServiceRegistrationServer(cpApiServer, cfg.PortRegistration)
	go grpc_helpers.CreateGRPCServer(utils.DockerLocalhost, cfg.Port, func(sr grpc.ServiceRegistrar) {
		proto.RegisterCpiInterfaceServer(sr, cpApiServer)
	})

	go profiler.SetupProfilerServer(cfg.Profiler)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		logrus.Info("Received interruption signal, try to gracefully stop")
	}
}

func parsePlacementPolicy(controlPlaneConfig config.ControlPlaneConfig) placement_policy.PlacementPolicy {
	switch controlPlaneConfig.PlacementPolicy {
	case "random":
		return placement_policy.NewRandomPlacement()
	case "round-robin":
		return placement_policy.NewRoundRobinPlacement()
	case "kubernetes":
		return placement_policy.NewKubernetesPolicy()
	default:
		logrus.Error("Failed to parse placement, default policy is random")
		return placement_policy.NewRandomPlacement()
	}
}
