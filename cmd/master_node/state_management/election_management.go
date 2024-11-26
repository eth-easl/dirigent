/*
 * MIT License
 *
 * Copyright (c) 2024 EASL
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package state_management

import (
	"cluster_manager/api"
	"cluster_manager/internal/control_plane/leader_election"
	"cluster_manager/pkg/config"
	"context"
	"github.com/sirupsen/logrus"
	"time"
)

type CurrentState struct {
	cpApiServer       *api.CpApiServer
	cpApiCreationArgs *api.CpApiServerCreationArguments
	cfg               config.ControlPlaneConfig

	stopNodeMonitoring chan struct{}
	wasLeaderBefore    bool
}

func NewElectionState(cfg config.ControlPlaneConfig, cpiApiServer *api.CpApiServer, cpApiCreationArgs *api.CpApiServerCreationArguments) *CurrentState {
	return &CurrentState{
		stopNodeMonitoring: nil,
		wasLeaderBefore:    false,

		cpApiServer:       cpiApiServer,
		cpApiCreationArgs: cpApiCreationArgs,
		cfg:               cfg,
	}
}

func (electionState *CurrentState) SetCurrentControlPlaneLeader(_ leader_election.AnnounceLeadership) {
	electionState.destroyStateFromPreviousElectionTerm()

	electionState.reconstructControlPlaneState()

	electionState.stopNodeMonitoring = electionState.cpApiServer.StartNodeMonitoringLoop()

	electionState.wasLeaderBefore = true
}

func (electionState *CurrentState) UpdateLeadership(leadership leader_election.AnnounceLeadership) {
	if leadership.IsLeader && !electionState.wasLeaderBefore {
		electionState.SetCurrentControlPlaneLeader(leadership)
		go electionState.cpApiServer.ControlPlane.ColdStartTracing.StartTracingService()
		logrus.Infof("Proceeding as the leader for the term #%d...", leadership.Term)
	} else {
		if electionState.wasLeaderBefore {
			electionState.destroyStateFromPreviousElectionTerm()
			electionState.stopNodeMonitoring = nil
		}
		electionState.wasLeaderBefore = false

		logrus.Infof("Another node was elected as the leader. Proceeding as a follower...")
	}
}

func (electionState *CurrentState) destroyStateFromPreviousElectionTerm() {
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
