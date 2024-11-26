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

// LeaderElectionServer container for a Raft Consensus Module. Exposes Raft to the network
// and enables RPCs between Raft peers.

import (
	"cluster_manager/api/proto"
	"cluster_manager/pkg/grpc_helpers"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type AnnounceLeadership struct {
	IsLeader bool
	Term     int
}

// LeaderElectionServer wraps a raft.ConsensusModule along with a rpc.Server that exposes its
// methods as RPC endpoints. It also manages the peers of the Raft server. The
// main goal of this type is to simplify the code of raft.LeaderElectionServer for
// presentation purposes. raft.ConsensusModule has a *LeaderElectionServer to do its peer
// communication and doesn't have to worry about the specifics of running an
// RPC server.
type LeaderElectionServer struct {
	mu sync.Mutex

	serverId int32
	peerIds  []int

	cm *ConsensusModule

	rpcServer *grpc.Server
	listener  net.Listener

	peerClients map[int]proto.CpiInterfaceClient

	ready <-chan interface{}
	quit  chan interface{}
	wg    sync.WaitGroup

	announceLeadership chan AnnounceLeadership
}

func NewServer(serverId int32, peerIds []int, ready <-chan interface{}) (*LeaderElectionServer, chan AnnounceLeadership) {
	s := new(LeaderElectionServer)
	s.serverId = serverId
	s.peerIds = peerIds
	s.peerClients = make(map[int]proto.CpiInterfaceClient)
	s.ready = ready
	s.quit = make(chan interface{})
	s.announceLeadership = make(chan AnnounceLeadership, 1)

	///////////////
	s.mu.Lock()
	s.cm = NewConsensusModule(s.serverId, s.peerIds, s, s.ready, s.announceLeadership)
	s.mu.Unlock()
	///////////////

	return s, s.announceLeadership
}

func (s *LeaderElectionServer) IsLeader() bool {
	return s.cm.IsLeader()
}

func (s *LeaderElectionServer) GetLeader() proto.CpiInterfaceClient {
	if s.cm.IsLeader() {
		logrus.Fatal("This function should not be called if you are the leader.")
	} else {
		leaderID := s.cm.GetLeaderID()

		s.mu.Lock()
		defer s.mu.Unlock()

		if api, ok := s.peerClients[leaderID]; ok {
			return api
		} else {
			logrus.Errorf("Forwarding request received before establishing leadership.")
		}
	}

	return nil
}

// DisconnectAll closes all the client connections to peers for this server.
func (s *LeaderElectionServer) DisconnectAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id := range s.peerClients {
		if _, ok := s.peerClients[id]; ok {
			delete(s.peerClients, id)
		}
	}
}

// Shutdown closes the server and waits for it to shut down properly.
func (s *LeaderElectionServer) Shutdown() {
	s.cm.Stop()
	close(s.quit)
	s.rpcServer.Stop()
	s.wg.Wait()
}

func (s *LeaderElectionServer) GetListenAddr() net.Addr {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.listener.Addr()
}

func (s *LeaderElectionServer) ConnectToPeer(peerId int, addr net.Addr) bool {
	conn := grpc_helpers.EstablishGRPCConnectionPoll([]string{addr.String()})
	if conn == nil {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.peerClients[peerId]; !ok {
		s.peerClients[peerId] = proto.NewCpiInterfaceClient(conn)
	}

	return true
}

// DisconnectPeer disconnects this server from the peer identified by peerId.
func (s *LeaderElectionServer) DisconnectPeer(peerId int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.peerClients[peerId]; ok {
		delete(s.peerClients, peerId)
	}
}

func sleepWithJitter() {
	time.Sleep(time.Duration(1+rand.Intn(5)) * time.Millisecond)
}

func (s *LeaderElectionServer) RequestVote(args *proto.RequestVoteArgs) (*proto.RequestVoteReply, error) {
	sleepWithJitter()
	return s.cm.RequestVote(args)
}

func (s *LeaderElectionServer) AppendEntries(args *proto.AppendEntriesArgs) (*proto.AppendEntriesReply, error) {
	sleepWithJitter()
	return s.cm.AppendEntries(args)
}

func (s *LeaderElectionServer) EstablishLeaderElectionMesh(replicas []string) {
	wg := sync.WaitGroup{}
	votesNeeded := int32(len(replicas)+1) / 2 // include myself
	votesCurrent := int32(0)

	wg.Add(int(votesNeeded))
	for _, rawAddress := range replicas {
		_, peerID := grpc_helpers.SplitAddress(rawAddress)
		tcpAddr, _ := net.ResolveTCPAddr("tcp", rawAddress)

		go func() {
			for {
				success := s.ConnectToPeer(peerID, tcpAddr)
				if success {
					newValue := atomic.AddInt32(&votesCurrent, 1)
					if newValue <= votesNeeded {
						wg.Done()
					}

					break
				}
			}
		}()
	}

	// signal that a connection with at least half ot the nodes has been established
	wg.Wait()
}

func (s *LeaderElectionServer) GetPeers() map[int]proto.CpiInterfaceClient {
	return s.peerClients
}
