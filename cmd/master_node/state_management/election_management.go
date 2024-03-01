package state_management

import (
	"cluster_manager/api"
	"cluster_manager/internal/control_plane/leader_election"
	"cluster_manager/internal/control_plane/registration_server"
	"cluster_manager/pkg/config"
	"context"
	"github.com/sirupsen/logrus"
	"net"
	"time"
)

type CurrentState struct {
	cpApiServer       *api.CpApiServer
	cpApiCreationArgs *api.CpApiServerCreationArguments
	cfg               config.ControlPlaneConfig

	stopNodeMonitoring     chan struct{}
	stopRegistrationServer chan struct{}
	wasLeaderBefore        bool
}

func NewElectionState(cfg config.ControlPlaneConfig, cpiApiServer *api.CpApiServer, cpApiCreationArgs *api.CpApiServerCreationArguments) *CurrentState {
	return &CurrentState{
		stopNodeMonitoring:     nil,
		stopRegistrationServer: nil,
		wasLeaderBefore:        false,

		cpApiServer:       cpiApiServer,
		cpApiCreationArgs: cpApiCreationArgs,
		cfg:               cfg,
	}
}

func (electionState *CurrentState) UpdateLeadership(leadership leader_election.AnnounceLeadership) {
	if leadership.IsLeader && !electionState.wasLeaderBefore {
		electionState.destroyStateFromPreviousElectionTerm()
		electionState.stopNodeMonitoring, electionState.stopRegistrationServer = nil, nil

		electionState.reconstructControlPlaneState()

		electionState.cpApiServer.ReviseHAProxyServers()
		electionState.stopNodeMonitoring = electionState.cpApiServer.StartNodeMonitoringLoop()

		_, registrationPort, err := net.SplitHostPort(electionState.cfg.RegistrationServer)
		if err != nil {
			logrus.Fatal("Invalid registration server address.")
		}

		_, electionState.stopRegistrationServer = registration_server.StartServiceRegistrationServer(electionState.cpApiServer, registrationPort, leadership.Term)
		electionState.cpApiServer.HAProxyAPI.ReviseRegistrationServers([]string{electionState.cfg.RegistrationServer})

		electionState.wasLeaderBefore = true
		logrus.Infof("Proceeding as the leader for the term #%d...", leadership.Term)
	} else {
		// make sure the HAProxy is stopped if not the leader
		electionState.cpApiServer.HAProxyAPI.StopHAProxy()

		if electionState.wasLeaderBefore {
			electionState.destroyStateFromPreviousElectionTerm()
			electionState.stopNodeMonitoring, electionState.stopRegistrationServer = nil, nil
		}
		electionState.wasLeaderBefore = false

		logrus.Infof("Another node was elected as the leader. Proceeding as a follower...")
	}
}

func (electionState *CurrentState) destroyStateFromPreviousElectionTerm() {
	if electionState.stopRegistrationServer != nil {
		electionState.stopRegistrationServer <- struct{}{}
	}

	if electionState.stopNodeMonitoring != nil {
		electionState.stopNodeMonitoring <- struct{}{}
	}

	electionState.cpApiServer.CleanControlPlaneInMemoryData(electionState.cpApiCreationArgs)
}

func (electionState *CurrentState) reconstructControlPlaneState() {
	start := time.Now()

	err := electionState.cpApiServer.ReconstructState(context.Background(), electionState.cfg)
	if err != nil {
		logrus.Fatalf("Failed to reconstruct state (error : %s)", err.Error())
	}

	elapsed := time.Since(start)
	logrus.Infof("Took %s to reconstruct", elapsed)
}
