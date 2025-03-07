package tracing

import (
	"cluster_manager/proto"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	coldStartLogHeader = "time,service_name,container_id,event,success,image_fetch,sandbox_create,sandbox_start,network_setup,iptables,readiness_probe,data_plane_propagation,snapshot_creation,configure_monitoring,find_snapshot,db,other_worker_node\n"
	proxyLogHeader     = "time,service_name,container_id,start_time,get_metadata,add_deployment,cold_start,load_balancing,cc_throttling,proxying,serialization,persistence_layer,other\n"
	taskLogHeader      = "time,task_name,container_url,start_time,total,get_metadata,add_deployment,start_worker,load_balancing,cc_throttling,execution,other,dandelion_traces\n"
	workflowLogHeader  = "time,service_name,start_time,total,get_metadata,preparation,execution,finalize,other,scheduler_startup,scheduler_scheduling,scheduler_executing,persistence_layer,serialization\n"
)

type ColdStartLogEntry struct {
	ServiceName     string
	ContainerID     string
	Event           string
	Success         bool
	PersistenceCost time.Duration

	LatencyBreakdown *proto.SandboxCreationBreakdown
}

type ProxyLogEntry struct {
	ServiceName string
	ContainerID string

	StartTime     time.Time
	Total         time.Duration
	GetMetadata   time.Duration
	AddDeployment time.Duration
	ColdStart     time.Duration
	LoadBalancing time.Duration
	CCThrottling  time.Duration
	Proxying      time.Duration

	PersistenceLayer time.Duration
	Serialization    time.Duration
}

type TaskLogEntry struct {
	TaskName     string // name of the task that is invoked
	ContainerUrl string // url of the worker chosen to run the task

	StartTime     time.Time     // timestamp when the invocation is received
	Total         time.Duration // end-to-end time spent working the task invocation
	GetMetadata   time.Duration // time spent fetching the service metadata
	AddDeployment time.Duration // time spent to register the task on the worker
	StartWorker   time.Duration // time spent to find a worker to execute the task
	LoadBalancing time.Duration // time spent load balancing
	CCThrottling  time.Duration // time spent concurrency throttling
	Execution     time.Duration // time from when the request is sent to the worker until a response is received

	DandelionTraces string // traces returned from the dandelion worker
}

type WorkflowLogEntry struct {
	ServiceName string // name of the service (workflow) that is invoked

	StartTime   time.Time     // timestamp when the invocation is received
	Total       time.Duration // end-to-end time spent working the workflow request
	GetMetadata time.Duration // time spent fetching the service metadata
	Preparation time.Duration // duration from starting to parse the request until execution starts
	Execution   time.Duration // end-to-end time executing the workflow
	Finalize    time.Duration // duration from the end of the execution until the response is ready to be returned

	SchedulerStartup    time.Duration // time spent setting up the scheduler
	SchedulerScheduling time.Duration // time spent deciding which task(s) to run next
	SchedulerExecuting  time.Duration // time spent in the scheduleTask function

	// async handler traces
	PersistenceLayer time.Duration
	Serialization    time.Duration
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
		InputChannel:  make(chan ColdStartLogEntry, 10000),
		Header:        coldStartLogHeader,
		WriteFunction: coldStartWriteFunction,
	}
}

func NewProxyTracingService(outputFile string) *TracingService[ProxyLogEntry] {
	return &TracingService[ProxyLogEntry]{
		OutputFile:    outputFile,
		InputChannel:  make(chan ProxyLogEntry, 10000),
		Header:        proxyLogHeader,
		WriteFunction: proxyWriteFunction,
	}
}

func NewTaskTracingService(outputFile string) *TracingService[TaskLogEntry] {
	return &TracingService[TaskLogEntry]{
		OutputFile:    outputFile,
		InputChannel:  make(chan TaskLogEntry, 10000),
		Header:        taskLogHeader,
		WriteFunction: taskWriteFunction,
	}
}

func NewWorkflowTracingService(outputFile string) *TracingService[WorkflowLogEntry] {
	return &TracingService[WorkflowLogEntry]{
		OutputFile:    outputFile,
		InputChannel:  make(chan WorkflowLogEntry, 10000),
		Header:        workflowLogHeader,
		WriteFunction: workflowWriteFunction,
	}
}
func CreateFileIfNotExist(path string) *os.File {
	directory := filepath.Dir(path)
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		if err := os.Mkdir(directory, 0777); err != nil {
			log.Fatal(err)
		}
	}

	os.Remove(path) // We don't want previous values
	f, err := os.Create(path)
	if err != nil {
		logrus.Fatalf("Unable to open output log file : %s", err.Error())
	}

	return f
}

func coldStartWriteFunction(f *os.File, msg ColdStartLogEntry) {
	if msg.LatencyBreakdown == nil {
		logrus.Errorf("No latency breakdown provided.")
		return
	}

	// DB should not be included in 'other' as 'LatencyBreakdown.Total' is only the worker node part
	other := msg.LatencyBreakdown.Total.AsDuration() - (msg.LatencyBreakdown.ImageFetch.AsDuration() +
		msg.LatencyBreakdown.SandboxCreate.AsDuration() + msg.LatencyBreakdown.SandboxStart.AsDuration() +
		msg.LatencyBreakdown.NetworkSetup.AsDuration() + msg.LatencyBreakdown.Iptables.AsDuration() +
		msg.LatencyBreakdown.ReadinessProbing.AsDuration() + msg.LatencyBreakdown.SnapshotCreation.AsDuration() +
		msg.LatencyBreakdown.ConfigureMonitoring.AsDuration() + msg.LatencyBreakdown.FindSnapshot.AsDuration())

	if _, err := f.WriteString(fmt.Sprintf("%d,%s,%s,%s,%t,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d\n",
		time.Now().UnixNano(),
		msg.ServiceName,
		msg.ContainerID,
		msg.Event,
		msg.Success,
		msg.LatencyBreakdown.ImageFetch.AsDuration().Microseconds(),
		msg.LatencyBreakdown.SandboxCreate.AsDuration().Microseconds(),
		msg.LatencyBreakdown.SandboxStart.AsDuration().Microseconds(),
		msg.LatencyBreakdown.NetworkSetup.AsDuration().Microseconds(),
		msg.LatencyBreakdown.Iptables.AsDuration().Microseconds(),
		msg.LatencyBreakdown.ReadinessProbing.AsDuration().Microseconds(),
		msg.LatencyBreakdown.DataplanePropagation.AsDuration().Microseconds(), // not part of other
		msg.LatencyBreakdown.SnapshotCreation.AsDuration().Microseconds(),
		msg.LatencyBreakdown.ConfigureMonitoring.AsDuration().Microseconds(),
		msg.LatencyBreakdown.FindSnapshot.AsDuration().Microseconds(),
		msg.PersistenceCost.Microseconds(),
		other.Microseconds(),
	)); err != nil {
		logrus.Errorf("Failed to write to file : %s", err.Error())
	}
}

func proxyWriteFunction(f *os.File, msg ProxyLogEntry) {
	other := msg.Total - (msg.GetMetadata + msg.AddDeployment + msg.ColdStart + msg.LoadBalancing + msg.CCThrottling + msg.Proxying + msg.PersistenceLayer + msg.Serialization)

	if _, err := f.WriteString(fmt.Sprintf("%d,%s,%s,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d\n",
		time.Now().UnixNano(),
		msg.ServiceName,
		msg.ContainerID,
		msg.StartTime.UnixNano(),
		msg.GetMetadata.Microseconds(),
		msg.AddDeployment.Microseconds(),
		msg.ColdStart.Microseconds(),
		msg.LoadBalancing.Microseconds(),
		msg.CCThrottling.Microseconds(),
		msg.Proxying.Microseconds(),
		msg.Serialization.Microseconds(),
		msg.PersistenceLayer.Microseconds(),
		other.Microseconds(),
	)); err != nil {
		logrus.Errorf("Failed to write to file : %s", err.Error())
	}
}

func taskWriteFunction(f *os.File, msg TaskLogEntry) {
	other := msg.Total - (msg.GetMetadata + msg.AddDeployment + msg.StartWorker + msg.LoadBalancing + msg.CCThrottling + msg.Execution)

	if _, err := f.WriteString(fmt.Sprintf("%d,%s,%s,%d,%d,%d,%d,%d,%d,%d,%d,%d,\"%s\"\n",
		time.Now().UnixNano(),
		msg.TaskName,
		msg.ContainerUrl,
		msg.StartTime.UnixNano(),
		msg.Total.Microseconds(),
		msg.GetMetadata.Microseconds(),
		msg.AddDeployment.Microseconds(),
		msg.StartWorker.Microseconds(),
		msg.LoadBalancing.Microseconds(),
		msg.CCThrottling.Microseconds(),
		msg.Execution.Microseconds(),
		other.Microseconds(),
		msg.DandelionTraces,
	)); err != nil {
		logrus.Errorf("Failed to write to file : %s", err.Error())
	}
}

func workflowWriteFunction(f *os.File, msg WorkflowLogEntry) {
	other := msg.Total - (msg.GetMetadata + msg.Preparation + msg.Execution + msg.Finalize)

	if _, err := f.WriteString(fmt.Sprintf("%d,%s,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d\n",
		time.Now().UnixNano(),
		msg.ServiceName,
		msg.StartTime.UnixNano(),
		msg.Total.Microseconds(),
		msg.GetMetadata.Microseconds(),
		msg.Preparation.Microseconds(),
		msg.Execution.Microseconds(),
		msg.Finalize.Microseconds(),
		other.Microseconds(),
		msg.SchedulerStartup.Microseconds(),
		msg.SchedulerScheduling.Microseconds(),
		msg.SchedulerExecuting.Microseconds(),
		msg.PersistenceLayer.Microseconds(),
		msg.Serialization.Microseconds(),
	)); err != nil {
		logrus.Errorf("Failed to write to file : %s", err.Error())
	}
}
