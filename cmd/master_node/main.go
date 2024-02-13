package main

import (
	"cluster_manager/api"
	"cluster_manager/internal/control_plane/data_plane"
	"cluster_manager/internal/control_plane/data_plane/empty_dataplane"
	"cluster_manager/internal/control_plane/persistence"
	"cluster_manager/internal/control_plane/placement_policy"
	"cluster_manager/internal/control_plane/registration_server"
	"cluster_manager/internal/control_plane/workers"
	"cluster_manager/internal/control_plane/workers/empty_worker"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/logger"
	"cluster_manager/pkg/profiler"
	"context"
	"flag"
	"os/signal"
	"path"
	"syscall"
	"time"

	_ "net/http/pprof"

	"github.com/sirupsen/logrus"
)

var (
	configPath = flag.String("config", "cmd/master_node/config_r1.yaml", "Path to the configuration file")
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

	/////////////////////////////////////////
	// COMMON FOR EVERY LEADER TERM
	/////////////////////////////////////////
	cpApiCreationArgs := &api.CpApiServerCreationArguments{
		Client:            persistenceLayer,
		OutputFile:        path.Join(cfg.TraceOutputFolder, "cold_start_trace.csv"),
		PlacementPolicy:   parsePlacementPolicy(cfg),
		DataplaneCreator:  dataplaneCreator,
		WorkerNodeCreator: workerNodeCreator,
		Cfg:               &cfg,
	}

	cpApiServer, isLeader := api.CreateNewCpApiServer(cpApiCreationArgs)

	go profiler.SetupProfilerServer(cfg.Profiler)

	go cpApiServer.ControlPlane.ColdStartTracing.StartTracingService()
	defer close(cpApiServer.ControlPlane.ColdStartTracing.InputChannel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	/////////////////////////////////////////
	// LEADERSHIP SPECIFIC
	/////////////////////////////////////////
	var stopNodeMonitoring chan struct{}
	var stopRegistrationServer chan struct{}

	wasLeaderBefore := false

	for {
		select {
		case leadership := <-isLeader:
			if leadership.IsLeader && !wasLeaderBefore {
				destroyStateFromPreviousElectionTerm(cpApiServer, cpApiCreationArgs, stopNodeMonitoring, stopRegistrationServer)
				stopNodeMonitoring, stopRegistrationServer = nil, nil

				ReconstructControlPlaneState(&cfg, cpApiServer)

				stopNodeMonitoring = cpApiServer.StartNodeMonitoringLoop()
				_, stopRegistrationServer = registration_server.StartServiceRegistrationServer(cpApiServer, cfg.PortRegistration, leadership.Term)

				wasLeaderBefore = true
				logrus.Infof("Proceeding as the leader for the term #%d...", leadership.Term)
			} else {
				if wasLeaderBefore {
					destroyStateFromPreviousElectionTerm(cpApiServer, cpApiCreationArgs, stopNodeMonitoring, stopRegistrationServer)
					stopNodeMonitoring, stopRegistrationServer = nil, nil
				}
				wasLeaderBefore = false

				logrus.Infof("Another node was elected as the leader. Proceeding as a follower...")
			}
		case <-ctx.Done():
			logrus.Info("Received interruption signal, try to gracefully stop")
			return
		}
	}
	/////////////////////////////////////////
	/////////////////////////////////////////
	/////////////////////////////////////////
}

func destroyStateFromPreviousElectionTerm(cpApiServer *api.CpApiServer, args *api.CpApiServerCreationArguments,
	stopNodeMonitoring chan struct{}, stopRegistrationServer chan struct{}) {

	if stopRegistrationServer != nil {
		stopRegistrationServer <- struct{}{}
	}

	if stopNodeMonitoring != nil {
		stopNodeMonitoring <- struct{}{}
	}

	cpApiServer.CleanControlPlaneInMemoryData(args)
}

func ReconstructControlPlaneState(cfg *config.ControlPlaneConfig, cpApiServer *api.CpApiServer) {
	start := time.Now()

	err := cpApiServer.ReconstructState(context.Background(), *cfg)
	if err != nil {
		logrus.Errorf("Failed to reconstruct state (error : %s)", err.Error())
	}

	elapsed := time.Since(start)
	logrus.Infof("Took %s to reconstruct", elapsed)
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
