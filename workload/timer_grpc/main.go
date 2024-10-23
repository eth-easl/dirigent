package main

import (
	"context"
	"fmt"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/reflection"
)

// static double SQRTSD (double x) {
//     double r;
//     __asm__ ("sqrtsd %1, %0" : "=x" (r) : "x" (x));
//     return r;
// }
import "C"

const (
	// ContainerImageSizeMB was chosen as a median of the container physical memory usage.
	// Allocate this much less memory inside the actual function.
	ContainerImageSizeMB = 15
)

const EXEC_UNIT int = 1e2

var hostname string
var IterationsMultiplier int
var serverSideCode FunctionType

type FunctionType int

const (
	TraceFunction FunctionType = 0
	EmptyFunction FunctionType = 1
)

func takeSqrts() C.double {
	var tmp C.double // Circumvent compiler optimizations
	for i := 0; i < EXEC_UNIT; i++ {
		tmp = C.SQRTSD(C.double(10))
	}
	return tmp
}

type funcServer struct {
	UnimplementedExecutorServer
}

func busySpin(runtimeMilli uint32) {
	totalIterations := IterationsMultiplier * int(runtimeMilli)

	for i := 0; i < totalIterations; i++ {
		takeSqrts()
	}
}

func TraceFunctionExecution(start time.Time, timeLeftMilliseconds uint32) (msg string) {
	timeConsumedMilliseconds := uint32(time.Since(start).Milliseconds())
	if timeConsumedMilliseconds < timeLeftMilliseconds {
		timeLeftMilliseconds -= timeConsumedMilliseconds
		if timeLeftMilliseconds > 0 {
			busySpin(timeLeftMilliseconds)
		}

		msg = fmt.Sprintf("OK - %s", hostname)
	}

	return msg
}

func (s *funcServer) Execute(_ context.Context, req *FaasRequest) (*FaasReply, error) {
	var msg string
	start := time.Now()

	if serverSideCode == TraceFunction {
		// Minimum execution time is AWS billing granularity - 1ms,
		// as defined in SpecificationGenerator::generateExecutionSpecs
		timeLeftMilliseconds := req.RuntimeInMilliSec
		/*toAllocate := util.Mib2b(req.MemoryInMebiBytes - ContainerImageSizeMB)
		if toAllocate < 0 {
			toAllocate = 0
		}*/

		// make is equivalent to `calloc` in C. The memory gets allocated
		// and zero is written to every byte, i.e. each page should be touched at least once
		//memory := make([]byte, toAllocate)
		// NOTE: the following statement to make sure the compiler does not treat the allocation as dead code
		//log.Debugf("Allocated memory size: %d\n", len(memory))
		msg = TraceFunctionExecution(start, timeLeftMilliseconds)
	} else {
		msg = fmt.Sprintf("OK - EMPTY - %s", hostname)
	}

	return &FaasReply{
		Message:            msg,
		DurationInMicroSec: uint32(time.Since(start).Microseconds()),
		MemoryUsageInKb:    req.MemoryInMebiBytes * 1024,
	}, nil
}

func readEnvironmentalVariables() {
	if _, ok := os.LookupEnv("ITERATIONS_MULTIPLIER"); ok {
		IterationsMultiplier, _ = strconv.Atoi(os.Getenv("ITERATIONS_MULTIPLIER"))
	} else {
		// Cloudlab xl170 benchmark @ 1 second function execution time
		IterationsMultiplier = 102
	}

	log.Infof("ITERATIONS_MULTIPLIER = %d\n", IterationsMultiplier)

	var err error
	hostname, err = os.Hostname()
	if err != nil {
		log.Warn("Failed to get HOSTNAME environmental variable.")
		hostname = "Unknown host"
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func StartHealthHandler(matcher net.Listener, port int) *http.Server {
	healthServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(healthHandler),
	}

	go func() {
		if err := healthServer.Serve(matcher); err != nil {
			log.Fatalf("Failed to start health handler. Terminating...")
		}
	}()

	return healthServer
}

func Start(serverAddress string, serverPort int, functionType FunctionType) {
	readEnvironmentalVariables()
	serverSideCode = functionType

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGTERM)

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", serverAddress, serverPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	m := cmux.New(lis)
	grpcMatcher := m.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	healthMatcher := m.Match(cmux.Any())

	healthServer := StartHealthHandler(healthMatcher, serverPort)

	grpcServer := grpc.NewServer()

	go func() {
		<-sigc
		log.Info("Received SIGTERM, shutting down gracefully...")

		healthServer.Shutdown(context.Background())
		grpcServer.GracefulStop()
	}()

	reflection.Register(grpcServer)
	RegisterExecutorServer(grpcServer, &funcServer{})
	go func() { err = grpcServer.Serve(grpcMatcher) }()

	m.Serve()
}

func main() {
	Start("0.0.0.0", 80, TraceFunction)
}
