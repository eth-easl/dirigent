package core

import "cluster_manager/pkg/tracing"

type DataplaneFactory func(string, string, string) DataPlaneInterface

type WorkerNodeConfiguration struct {
	Name     string
	IP       string
	Port     string
	CpuCores int
	Memory   int
}

type WorkerNodeFactory func(configuration WorkerNodeConfiguration) WorkerNodeInterface

type Endpoint struct {
	SandboxID       string
	URL             string
	Node            WorkerNodeInterface
	HostPort        int32
	CreationHistory tracing.ColdStartLogEntry
}
