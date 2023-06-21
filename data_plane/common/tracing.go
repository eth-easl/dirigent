package common

import (
	"cluster_manager/api/proto"
	"fmt"
	"github.com/sirupsen/logrus"
	"log"
	"os"
	"path/filepath"
	"time"
)

const (
	coldStartLogHeader = "time,service_name,container_id,success,image_fetch,container_create,container_start,cni,iptables,other\n"
	proxyLogHeader     = "time,service_name,container_id,get_metadata,cold_start,cold_start_pause,load_balancing,cc_throttling,proxying,other\n"
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

	Total          time.Duration
	GetMetadata    time.Duration
	ColdStart      time.Duration
	ColdStartPause time.Duration
	LoadBalancing  time.Duration
	CCThrottling   time.Duration
	Proxying       time.Duration
}

type TracingService[T any] struct {
	OutputFile   string
	InputChannel chan T

	Header        string
	WriteFunction func(*os.File, T)
}

func (ts *TracingService[K]) StartTracingService() {
	f := CreateFileIfNotExist(ts.OutputFile)
	defer f.Close()

	_, _ = f.WriteString(ts.Header)

	for {
		msg, ok := <-ts.InputChannel
		if !ok {
			break
		}

		ts.WriteFunction(f, msg)
	}
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

	f, err := os.Create(path)
	if err != nil {
		logrus.Fatal("Unable to open output log file.")
	}

	return f
}

func coldStartWriteFunction(f *os.File, msg ColdStartLogEntry) {
	other := msg.LatencyBreakdown.Total.AsDuration() - (msg.LatencyBreakdown.ImageFetch.AsDuration() +
		msg.LatencyBreakdown.ContainerCreate.AsDuration() + msg.LatencyBreakdown.ContainerStart.AsDuration() +
		msg.LatencyBreakdown.CNI.AsDuration() + msg.LatencyBreakdown.Iptables.AsDuration())

	_, _ = f.WriteString(fmt.Sprintf("%d,%s,%s,%t,%d,%d,%d,%d,%d,%d\n",
		time.Now().Nanosecond(),
		msg.ContainerID,
		msg.ServiceName,
		msg.Success,
		msg.LatencyBreakdown.ImageFetch.AsDuration().Microseconds(),
		msg.LatencyBreakdown.ContainerCreate.AsDuration().Microseconds(),
		msg.LatencyBreakdown.ContainerStart.AsDuration().Microseconds(),
		msg.LatencyBreakdown.CNI.AsDuration().Microseconds(),
		msg.LatencyBreakdown.Iptables.AsDuration().Microseconds(),
		other.Microseconds(),
	))
}

func proxyWriteFunction(f *os.File, msg ProxyLogEntry) {
	other := msg.Total - (msg.GetMetadata + msg.ColdStart + msg.ColdStartPause + msg.LoadBalancing + msg.CCThrottling + msg.Proxying)

	_, _ = f.WriteString(fmt.Sprintf("%d,%s,%s,%d,%d,%d,%d,%d,%d,%d\n",
		time.Now().Nanosecond(),
		msg.ServiceName,
		msg.ContainerID,
		msg.GetMetadata.Microseconds(),
		msg.ColdStart.Microseconds(),
		msg.ColdStartPause.Microseconds(),
		msg.LoadBalancing.Microseconds(),
		msg.CCThrottling.Microseconds(),
		msg.Proxying.Microseconds(),
		other.Microseconds(),
	))
}
