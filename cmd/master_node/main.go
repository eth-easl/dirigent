package main

import (
	"cluster_manager/api"
	"cluster_manager/cmd/master_node/state_management"
	"cluster_manager/internal/control_plane"
	"cluster_manager/internal/control_plane/data_plane"
	"cluster_manager/internal/control_plane/data_plane/empty_dataplane"
	"cluster_manager/internal/control_plane/persistence"
	"cluster_manager/internal/control_plane/registration_server"
	"cluster_manager/internal/control_plane/workers"
	"cluster_manager/internal/control_plane/workers/empty_worker"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/logger"
	"cluster_manager/pkg/profiler"
	"cluster_manager/pkg/utils"
	"context"
	"flag"
	"net"
	_ "net/http/pprof"
	"os/signal"
	"path"
	"syscall"

	"github.com/sirupsen/logrus"
)

var (
	configPath = flag.String("config", "cmd/master_node/config.yaml", "Path to the configuration file")
)

func main() {
	if !utils.IsRoot() {
		logrus.Fatalf("Master node must be started with sudo")
	}

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

	cpApiCreationArgs := &api.CpApiServerCreationArguments{
		Client:            persistenceLayer,
		OutputFile:        path.Join(cfg.TraceOutputFolder, "cold_start_trace.csv"),
		PlacementPolicy:   control_plane.ParsePlacementPolicy(cfg),
		DataplaneCreator:  dataplaneCreator,
		WorkerNodeCreator: workerNodeCreator,
		Cfg:               &cfg,
	}

	cpApiServer, isLeader := api.CreateNewCpApiServer(cpApiCreationArgs)
	stopRegistrationServer := registration_server.StartServiceRegistrationServer(cpApiServer, getRegistrationPort(&cfg))

	cpApiServer.HAProxyAPI.StartHAProxy()

	go profiler.SetupProfilerServer(cfg.Profiler)

	go cpApiServer.ControlPlane.ColdStartTracing.StartTracingService()
	defer close(cpApiServer.ControlPlane.ColdStartTracing.InputChannel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	electionState := state_management.NewElectionState(cfg, cpApiServer, cpApiCreationArgs)

	for {
		select {
		case leadership := <-isLeader:
			electionState.UpdateLeadership(leadership)
		case <-ctx.Done():
			stopRegistrationServer <- struct{}{}

			logrus.Info("Received interruption signal, try to gracefully stop")
			return
		}
	}
}

func getRegistrationPort(cfg *config.ControlPlaneConfig) string {
	_, registrationPort, err := net.SplitHostPort(cfg.RegistrationServer)
	if err != nil {
		logrus.Fatal("Invalid registration server address.")
	}

	return registrationPort
}
