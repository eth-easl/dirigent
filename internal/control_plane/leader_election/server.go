package leader_election

// LeaderElectionServer container for a Raft Consensus Module. Exposes Raft to the network
// and enables RPCs between Raft peers.
//
// Eli Bendersky [https://eli.thegreenplace.net]
// This code is in the public domain.

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
	//rpcProxy *RPCProxy

	rpcServer *grpc.Server
	listener  net.Listener

	peerClients map[int]proto.CpiInterfaceClient

	ready <-chan interface{}
	quit  chan interface{}
	wg    sync.WaitGroup

	announceLeadership chan bool
}

func NewServer(serverId int32, peerIds []int, ready <-chan interface{}) (*LeaderElectionServer, chan bool) {
	s := new(LeaderElectionServer)
	s.serverId = serverId
	s.peerIds = peerIds
	s.peerClients = make(map[int]proto.CpiInterfaceClient)
	s.ready = ready
	s.quit = make(chan interface{})
	s.announceLeadership = make(chan bool)

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

func (s *LeaderElectionServer) Serve(port int) {
	/*s.mu.Lock()
	s.cm = NewConsensusModule(s.serverId, s.peerIds, s, s.ready)

	// Create a new RPC server and register a RPCProxy that forwards all methods to n.cm
	s.rpcServer = grpc.NewServer()
	proto.RegisterRAFTInterfaceServer(s.rpcServer, &RPCProxy{cm: s.cm})

	var err error
	s.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("[%v] listening at %s", s.serverId, s.listener.Addr())
	s.mu.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		if err := s.rpcServer.Serve(s.listener); err != nil {
			log.Fatalf("[%d] failed to serve: %v", s.serverId, err)
		}
	}()*/
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

func (s *LeaderElectionServer) ConnectToPeer(peerId int, addr net.Addr) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.peerClients[peerId] == nil {
		address, port, _ := net.SplitHostPort(addr.String())
		conn := grpc_helpers.EstablishGRPCConnectionPoll(address, port)
		s.peerClients[peerId] = proto.NewCpiInterfaceClient(conn)
	}

	return nil
}

// DisconnectPeer disconnects this server from the peer identified by peerId.
func (s *LeaderElectionServer) DisconnectPeer(peerId int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.peerClients[peerId] != nil {
		delete(s.peerClients, peerId)
	}
	return nil
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
	/*if len(os.Getenv("RAFT_UNRELIABLE_RPC")) > 0 {
		dice := rand.Intn(10)
		if dice == 9 {
			rpp.cm.dlog("drop RequestVote")
		} else if dice == 8 {
			rpp.cm.dlog("delay RequestVote")
			time.Sleep(75 * time.Millisecond)
		}
	} else {*/
	time.Sleep(time.Duration(1+rand.Intn(5)) * time.Millisecond)
	//}

	return s.cm.RequestVote(args)
}

func (s *LeaderElectionServer) AppendEntries(args *proto.AppendEntriesArgs) (*proto.AppendEntriesReply, error) {
	/*if len(os.Getenv("RAFT_UNRELIABLE_RPC")) > 0 {
		dice := rand.Intn(10)
		if dice == 9 {
			rpp.cm.dlog("drop AppendEntries")
		} else if dice == 8 {
			rpp.cm.dlog("delay AppendEntries")
			time.Sleep(75 * time.Millisecond)
		}
	} else {*/
	time.Sleep(time.Duration(1+rand.Intn(5)) * time.Millisecond)
	//}

	return s.cm.AppendEntries(args)
}
