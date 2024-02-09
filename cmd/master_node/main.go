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
	"net/http"
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
	cpApiServer, isLeader := api.CreateNewCpApiServer(persistenceLayer, path.Join(cfg.TraceOutputFolder, "cold_start_trace.csv"), parsePlacementPolicy(cfg), dataplaneCreator, workerNodeCreator, &cfg)

	go profiler.SetupProfilerServer(cfg.Profiler)

	go cpApiServer.ControlPlane.ColdStartTracing.StartTracingService()
	defer close(cpApiServer.ControlPlane.ColdStartTracing.InputChannel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	/////////////////////////////////////////
	// LEADERSHIP SPECIFIC
	/////////////////////////////////////////
	var registrationServer *http.Server
	var stopNodeMonitoring chan struct{}

	for {
		select {
		case leader := <-isLeader:
			if leader {
				logrus.Infof("Proceeding as the leader for the current term...")

				// TODO: clear all the state from the previous leader election terms in the control plane
				// TODO: same holds for all the other go routines
				destroyStateFromPreviousElectionTerm(registrationServer, stopNodeMonitoring)
				//ReconstructControlPlaneState(&cfg, cpApiServer)

				cpApiServer.StartNodeMonitoringLoop(stopNodeMonitoring)
				registrationServer = registration_server.StartServiceRegistrationServer(cpApiServer, cfg.PortRegistration)
			} else {
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

func destroyStateFromPreviousElectionTerm(registrationServer *http.Server, stopNodeMonitoring chan struct{}) {
	if registrationServer != nil {
		logrus.Infof("Shutting down function registration server from the previous leader's term.")

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		if err := registrationServer.Shutdown(ctx); err != nil {
			logrus.Errorf("Failed to shut down function registration server.")
		}
	}

	if stopNodeMonitoring != nil {
		logrus.Infof("Stopping node monitoring from the previous leader's term.")

		stopNodeMonitoring <- struct{}{}
	}
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
