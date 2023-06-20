package common

import (
	"cluster_manager/api/proto"
	"fmt"
	"github.com/sirupsen/logrus"
	"log"
	"os"
	"path/filepath"
)

const (
	coldStartLogHeader = "container_id,success,image_fetch,container_create,container_start,cni,iptables,other\n"
	dataPlaneLogHeader = ""
)

type ColdStartLogEntry struct {
	ContainerID      string
	Success          bool
	LatencyBreakdown *proto.SandboxCreationBreakdown
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
		InputChannel:  make(chan ColdStartLogEntry),
		Header:        coldStartLogHeader,
		WriteFunction: coldStartWriteFunction,
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
	other := msg.LatencyBreakdown.ImageFetch.AsDuration() +
		msg.LatencyBreakdown.ContainerCreate.AsDuration() +
		msg.LatencyBreakdown.ContainerStart.AsDuration() +
		msg.LatencyBreakdown.CNI.AsDuration() +
		msg.LatencyBreakdown.Iptables.AsDuration()

	_, _ = f.WriteString(fmt.Sprintf("%s,%t,%d,%d,%d,%d,%d,%d\n",
		msg.ContainerID,
		msg.Success,
		msg.LatencyBreakdown.ImageFetch.AsDuration().Microseconds(),
		msg.LatencyBreakdown.ContainerCreate.AsDuration().Microseconds(),
		msg.LatencyBreakdown.ContainerStart.AsDuration().Microseconds(),
		msg.LatencyBreakdown.CNI.AsDuration().Microseconds(),
		msg.LatencyBreakdown.Iptables.AsDuration().Microseconds(),
		msg.LatencyBreakdown.Total.AsDuration().Microseconds()-other.Microseconds(),
	))
}
