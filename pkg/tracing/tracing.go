package tracing

import (
	"cluster_manager/api/proto"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	coldStartLogHeader = "time,service_name,container_id,success,image_fetch,sandbox_create,sandbox_start,network_setup,iptables,readiness_probe,data_plane_propagation,other_worker_node\n"
	proxyLogHeader     = "time,service_name,container_id,get_metadata,cold_start,load_balancing,cc_throttling,proxying,other\n"
)

type ColdStartLogEntry struct {
	ServiceName string
	ContainerID string
	Success     bool

	LatencyBreakdown *proto.SandboxCreationBreakdown
}

type ProxyLogEntry struct {
	ServiceName string
	ContainerID string

	Total         time.Duration
	GetMetadata   time.Duration
	ColdStart     time.Duration
	LoadBalancing time.Duration
	CCThrottling  time.Duration
	Proxying      time.Duration
}

type TracingService[T any] struct {
	OutputFile   string
	InputChannel chan T

	Header        string
	WriteFunction func(*os.File, T)
}

func (ts *TracingService[K]) StartTracingService() {
	f := CreateFileIfNotExist(ts.OutputFile)

	_, _ = f.WriteString(ts.Header)

	f.Close()

	for {
		msg, ok := <-ts.InputChannel
		if !ok {
			break
		}

		f, err := os.OpenFile(ts.OutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}

		ts.WriteFunction(f, msg)

		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	}
}

func (ts *TracingService[K]) ResetTracingService() {
	f := CreateFileIfNotExist(ts.OutputFile)
	defer f.Close()

	_, _ = f.WriteString(ts.Header)
}

func NewColdStartTracingService(outputFile string) *TracingService[ColdStartLogEntry] {
	return &TracingService[ColdStartLogEntry]{
		OutputFile:    outputFile,
		InputChannel:  make(chan ColdStartLogEntry, 100),
		Header:        coldStartLogHeader,
		WriteFunction: coldStartWriteFunction,
	}
}

func NewProxyTracingService(outputFile string) *TracingService[ProxyLogEntry] {
	return &TracingService[ProxyLogEntry]{
		OutputFile:    outputFile,
		InputChannel:  make(chan ProxyLogEntry, 100),
		Header:        proxyLogHeader,
		WriteFunction: proxyWriteFunction,
	}
}

func CreateFileIfNotExist(path string) *os.File {
	directory := filepath.Dir(path)
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		err = os.MkdirAll(directory, 0700)
		if err != nil {
			log.Fatal(err)
		}
	}

	os.Remove(path) // We don't want previous values
	f, err := os.Create(path)
	if err != nil {
		logrus.Fatal("Unable to open output log file.")
	}

	return f
}

func coldStartWriteFunction(f *os.File, msg ColdStartLogEntry) {
	// DB should not be included in 'other' as 'LatencyBreakdown.Total' is only the worker node part
	other := msg.LatencyBreakdown.Total.AsDuration() - (msg.LatencyBreakdown.ImageFetch.AsDuration() +
		msg.LatencyBreakdown.SandboxCreate.AsDuration() + msg.LatencyBreakdown.SandboxStart.AsDuration() +
		msg.LatencyBreakdown.NetworkSetup.AsDuration() + msg.LatencyBreakdown.Iptables.AsDuration() +
		msg.LatencyBreakdown.ReadinessProbing.AsDuration() + msg.LatencyBreakdown.DataplanePropagation.AsDuration())

	_, _ = f.WriteString(fmt.Sprintf("%d,%s,%s,%t,%d,%d,%d,%d,%d,%d,%d,%d\n",
		time.Now().UnixNano(),
		msg.ServiceName,
		msg.ContainerID,
		msg.Success,
		msg.LatencyBreakdown.ImageFetch.AsDuration().Microseconds(),
		msg.LatencyBreakdown.SandboxCreate.AsDuration().Microseconds(),
		msg.LatencyBreakdown.SandboxStart.AsDuration().Microseconds(),
		msg.LatencyBreakdown.NetworkSetup.AsDuration().Microseconds(),
		msg.LatencyBreakdown.Iptables.AsDuration().Microseconds(),
		msg.LatencyBreakdown.ReadinessProbing.AsDuration().Microseconds(),
		msg.LatencyBreakdown.DataplanePropagation.AsDuration().Microseconds(),
		other.Microseconds(),
	))
}

func proxyWriteFunction(f *os.File, msg ProxyLogEntry) {
	other := msg.Total - (msg.GetMetadata + msg.ColdStart + msg.LoadBalancing + msg.CCThrottling + msg.Proxying)

	_, _ = f.WriteString(fmt.Sprintf("%d,%s,%s,%d,%d,%d,%d,%d,%d\n",
		time.Now().UnixNano(),
		msg.ServiceName,
		msg.ContainerID,
		msg.GetMetadata.Microseconds(),
		msg.ColdStart.Microseconds(),
		msg.LoadBalancing.Microseconds(),
		msg.CCThrottling.Microseconds(),
		msg.Proxying.Microseconds(),
		other.Microseconds(),
	))
}
