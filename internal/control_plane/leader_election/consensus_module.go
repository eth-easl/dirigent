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

package leader_election

// Core Raft implementation - Consensus Module.
//
// Eli Bendersky [https://eli.thegreenplace.net]
// This code is in the public domain.

import (
	"cluster_manager/api/proto"
	"context"
	"github.com/sirupsen/logrus"
	"math/rand"
	"os"
	"sync"
	"time"
)

// ConsensusModule (CM) implements a single node of Raft consensus.
type ConsensusModule struct {
	// mu protects concurrent access to a CM.
	mu sync.Mutex

	// id is the server ID of this CM.
	id int32

	// peerIds lists the IDs of our peers in the cluster.
	peerIds []int

	// server is the server containing this CM. It's used to issue RPC calls
	// to peers.
	server *LeaderElectionServer

	// Persistent Raft state on all servers
	currentTerm int32
	votedFor    int32

	// Volatile Raft state on all servers
	state              CMState
	electionResetEvent time.Time

	// Dirigent-specific
	currentLeaderID    int32
	announceLeadership chan AnnounceLeadership
}

// NewConsensusModule creates a new CM with the given ID, list of peer IDs and
// server. The ready channel signals the CM that all peers are connected and
// it's safe to start its state machine.
func NewConsensusModule(id int32, peerIds []int, server *LeaderElectionServer, ready <-chan interface{}, announceLeadership chan AnnounceLeadership) *ConsensusModule {
	cm := new(ConsensusModule)
	cm.id = id
	cm.peerIds = peerIds
	cm.server = server
	cm.state = Follower
	cm.votedFor = -1
	cm.announceLeadership = announceLeadership

	if len(cm.peerIds) == 0 {
		cm.announceLeadership <- AnnounceLeadership{
			IsLeader: true,
			Term:     int(cm.currentTerm),
		}
		cm.state = Leader
		logrus.Infof("No peers found for the leader election. Proclaiming myself as the supreme leader...")
	}

	go func() {
		// The CM is quiescent until ready is signaled; then, it starts a countdown
		// for leader election.
		<-ready
		cm.mu.Lock()
		cm.electionResetEvent = time.Now()
		cm.mu.Unlock()
		cm.runElectionTimer()
	}()

	return cm
}

// Report reports the state of this CM.
func (cm *ConsensusModule) Report() (id int32, term int32, isLeader bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	return cm.id, cm.currentTerm, cm.state == Leader
}

// Stop stops this CM, cleaning up its state. This method returns quickly, but
// it may take a bit of time (up to ~election timeout) for all goroutines to
// exit.
func (cm *ConsensusModule) Stop() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.state = Dead
	logrus.Debug("Becomes Dead")
}

func (cm *ConsensusModule) IsLeader() bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	return cm.state == Leader
}

// RequestVote RPC.
func (cm *ConsensusModule) RequestVote(args *proto.RequestVoteArgs) (*proto.RequestVoteReply, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.state == Dead {
		return nil, nil
	}
	logrus.Tracef("RequestVote: %+v [currentTerm=%d, votedFor=%d]", args, cm.currentTerm, cm.votedFor)

	if args.Term > cm.currentTerm {
		logrus.Trace("... term out of date in RequestVote")
		cm.becomeFollower(args.Term)
	}

	reply := &proto.RequestVoteReply{}

	if cm.currentTerm == args.Term &&
		(cm.votedFor == -1 || cm.votedFor == args.CandidateID) {
		reply.VoteGranted = true
		cm.votedFor = args.CandidateID
		cm.electionResetEvent = time.Now()
	} else {
		reply.VoteGranted = false
	}
	reply.Term = cm.currentTerm
	logrus.Tracef("... RequestVote reply: %+v", reply)
	return reply, nil
}

func (cm *ConsensusModule) AppendEntries(args *proto.AppendEntriesArgs) (*proto.AppendEntriesReply, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.state == Dead {
		return nil, nil
	}

	if args.Term > cm.currentTerm {
		logrus.Tracef("... term out of date in AppendEntries")
		cm.becomeFollower(args.Term)
	}

	cm.currentLeaderID = args.LeaderID

	reply := &proto.AppendEntriesReply{}
	reply.Success = false
	if args.Term == cm.currentTerm {
		if cm.state != Follower {
			cm.becomeFollower(args.Term)
		}
		cm.electionResetEvent = time.Now()
		reply.Success = true
	}

	reply.Term = cm.currentTerm
	//logrus.Tracef("Received leader election heartbeat: %+v", reply)

	return reply, nil
}

func (cm *ConsensusModule) GetLeaderID() int {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	return int(cm.currentLeaderID)
}

// electionTimeout generates a pseudo-random election timeout duration.
func (cm *ConsensusModule) electionTimeout() time.Duration {
	// If RAFT_FORCE_MORE_REELECTION is set, stress-test by deliberately
	// generating a hard-coded number very often. This will create collisions
	// between different servers and force more re-elections.
	if len(os.Getenv("RAFT_FORCE_MORE_REELECTION")) > 0 && rand.Intn(3) == 0 {
		return time.Duration(150) * time.Millisecond
	} else {
		return time.Duration(150+rand.Intn(150)) * time.Millisecond
	}
}

// runElectionTimer implements an election timer. It should be launched whenever
// we want to start a timer towards becoming a candidate in a new election.
//
// This function is blocking and should be launched in a separate goroutine;
// it's designed to work for a single (one-shot) election timer, as it exits
// whenever the CM state changes from follower/candidate or the term changes.
func (cm *ConsensusModule) runElectionTimer() {
	timeoutDuration := cm.electionTimeout()
	cm.mu.Lock()
	termStarted := cm.currentTerm
	cm.mu.Unlock()
	logrus.Tracef("election timer started (%v), term=%d", timeoutDuration, termStarted)

	// This loops until either:
	// - we discover the election timer is no longer needed, or
	// - the election timer expires and this CM becomes a candidate
	// In a follower, this typically keeps running in the background for the
	// duration of the CM's lifetime.
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		<-ticker.C

		cm.mu.Lock()
		if cm.state != Candidate && cm.state != Follower {
			logrus.Tracef("in election timer state=%s, bailing out", cm.state)
			cm.mu.Unlock()
			return
		}

		if termStarted != cm.currentTerm {
			logrus.Tracef("in election timer term changed from %d to %d, bailing out", termStarted, cm.currentTerm)
			cm.mu.Unlock()
			return
		}

		// Start an election if we haven't heard from a leader or haven't voted for
		// someone for the duration of the timeout.
		if elapsed := time.Since(cm.electionResetEvent); elapsed >= timeoutDuration {
			cm.startElection()
			cm.mu.Unlock()
			return
		}
		cm.mu.Unlock()
	}
}

// startElection starts a new election with this CM as a candidate.
// Expects cm.mu to be locked.
func (cm *ConsensusModule) startElection() {
	cm.state = Candidate
	cm.currentTerm += 1
	savedCurrentTerm := cm.currentTerm
	cm.electionResetEvent = time.Now()
	cm.votedFor = cm.id
	logrus.Debugf("Becomes Candidate for term #%d", savedCurrentTerm)

	votesReceived := 1

	// Send RequestVote RPCs to all other servers concurrently.
	for _, peerId := range cm.peerIds {
		go func(peerId int) {
			args := &proto.RequestVoteArgs{
				Term:        savedCurrentTerm,
				CandidateID: cm.id,
			}

			logrus.Tracef("sending RequestVote to %d: %+v", peerId, args)

			client, ok := cm.server.peerClients[peerId]
			if !ok {
				logrus.Debugf("Client entry not found %d - voting", peerId)
				return
			}

			if reply, err := client.RequestVote(context.Background(), args); err == nil {
				cm.mu.Lock()
				defer cm.mu.Unlock()
				logrus.Tracef("received RequestVoteReply %+v", reply)

				if cm.state != Candidate {
					logrus.Tracef("while waiting for reply, state = %v", cm.state)
					return
				}

				if reply.Term > savedCurrentTerm {
					logrus.Debugf("term out of date in RequestVoteReply")
					cm.becomeFollower(reply.Term)
					return
				} else if reply.Term == savedCurrentTerm {
					if reply.VoteGranted {
						votesReceived += 1
						if votesReceived*2 > len(cm.peerIds)+1 {
							// Won the election!
							logrus.Debugf("Wins election for term #%d with %d votes", savedCurrentTerm, votesReceived)
							cm.startLeader()
							return
						}
					}
				}
			} else {
				logrus.Errorf("Connection with %d has been lost...", peerId)
			}

		}(peerId)
	}

	// Run another election timer, in case this election is not successful.
	go cm.runElectionTimer()
}

// becomeFollower makes cm a follower and resets its state.
// Expects cm.mu to be locked.
func (cm *ConsensusModule) becomeFollower(term int32) {
	logrus.Debugf("Becomes Follower for term #%d", term)

	cm.announceLeadership <- AnnounceLeadership{
		IsLeader: false,
		Term:     int(cm.currentTerm),
	}

	cm.state = Follower
	cm.currentTerm = term
	cm.votedFor = -1
	cm.electionResetEvent = time.Now()

	go cm.runElectionTimer()
}

// startLeader switches cm into a leader state and begins process of heartbeats.
// Expects cm.mu to be locked.
func (cm *ConsensusModule) startLeader() {
	cm.state = Leader
	cm.announceLeadership <- AnnounceLeadership{
		IsLeader: true,
		Term:     int(cm.currentTerm),
	}
	logrus.Debugf("Becomes Leader for term #%d", cm.currentTerm)

	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		// Send periodic heartbeats, as long as still leader.
		for {
			cm.leaderSendHeartbeats()
			<-ticker.C

			cm.mu.Lock()
			if cm.state != Leader {
				cm.mu.Unlock()
				return
			}
			cm.mu.Unlock()
		}
	}()
}

// leaderSendHeartbeats sends a round of heartbeats to all peers, collects their
// replies and adjusts cm's state.
func (cm *ConsensusModule) leaderSendHeartbeats() {
	cm.mu.Lock()
	if cm.state != Leader {
		cm.mu.Unlock()
		return
	}
	savedCurrentTerm := cm.currentTerm
	cm.mu.Unlock()

	for _, peerId := range cm.peerIds {
		args := &proto.AppendEntriesArgs{
			Term:     savedCurrentTerm,
			LeaderID: cm.id,
		}
		go func(peerId int) {
			//logrus.Tracef("Sending leader election heartbeat to %v: ni=%d, args=%+v", peerId, 0, args)
			client, ok := cm.server.peerClients[peerId]
			if !ok {
				//logrus.Debugf("Client entry not found %d - heartbeat", peerId)
				return
			}

			if reply, err := client.AppendEntries(context.Background(), args); err == nil {
				cm.mu.Lock()
				defer cm.mu.Unlock()
				if reply.Term > savedCurrentTerm {
					logrus.Debugf("term out of date in heartbeat reply")
					cm.becomeFollower(reply.Term)
					return
				}
			}
		}(peerId)
	}
}
