package leader_election

// LeaderElectionServer container for a Raft Consensus Module. Exposes Raft to the network
// and enables RPCs between Raft peers.

import (
	"cluster_manager/api/proto"
	"cluster_manager/pkg/grpc_helpers"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"math/rand"
	"net"
	"sync"
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
		api, ok := s.peerClients[leaderID]

		if ok {
			return api
		} else {
			logrus.Fatal("Invalid leader ID to obtain CpiInterfaceClient.")
		}
	}

	return nil
}

// DisconnectAll closes all the client connections to peers for this server.
func (s *LeaderElectionServer) DisconnectAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id := range s.peerClients {
		if s.peerClients[id] != nil {
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

func (s *LeaderElectionServer) ConnectToPeer(peerId int, addr net.Addr) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.peerClients[peerId] == nil {
		conn := grpc_helpers.EstablishGRPCConnectionPoll([]string{addr.String()})
		s.peerClients[peerId] = proto.NewCpiInterfaceClient(conn)
	}
}

// DisconnectPeer disconnects this server from the peer identified by peerId.
func (s *LeaderElectionServer) DisconnectPeer(peerId int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.peerClients[peerId] != nil {
		delete(s.peerClients, peerId)
	}
}

func (s *LeaderElectionServer) Call(id int, serviceMethod string, args interface{}, reply interface{}) error {
	s.mu.Lock()
	peer := s.peerClients[id]
	s.mu.Unlock()

	// If this is called after shutdown (where client.Close is called), it will
	// return an error.
	if peer == nil {
		return fmt.Errorf("call client %d after it's closed", id)
	} else {
		switch serviceMethod {
		case "ConsensusModule.RequestVote":
			r, err := peer.RequestVote(context.Background(), args.(*proto.RequestVoteArgs))
			reply = r
			return err
		case "ConsensusModule.AppendEntries":
			r, err := peer.AppendEntries(context.Background(), args.(*proto.AppendEntriesArgs))
			reply = r
			return err
		default:
			logrus.Fatal("Unsupported gRPC method.")
		}
	}

	return nil
}

func (s *LeaderElectionServer) RequestVote(args *proto.RequestVoteArgs) (*proto.RequestVoteReply, error) {
	time.Sleep(time.Duration(1+rand.Intn(5)) * time.Millisecond)

	return s.cm.RequestVote(args)
}

func (s *LeaderElectionServer) AppendEntries(args *proto.AppendEntriesArgs) (*proto.AppendEntriesReply, error) {
	time.Sleep(time.Duration(1+rand.Intn(5)) * time.Millisecond)

	return s.cm.AppendEntries(args)
}
