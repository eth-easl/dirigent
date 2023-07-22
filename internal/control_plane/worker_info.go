package control_plane

import (
	"cluster_manager/api/proto"
	"cluster_manager/pkg/grpc_helpers"
	"time"
)

type WorkerNode struct {
	Name string
	IP   string
	Port string

	CpuUsage    int
	MemoryUsage int

	CpuCores int
	Memory   int

	LastHeartbeat time.Time
	api           proto.WorkerNodeInterfaceClient
}

func (w *WorkerNode) GetAPI() proto.WorkerNodeInterfaceClient {
	if w.api == nil {
		w.api = grpc_helpers.InitializeWorkerNodeConnection(w.IP, w.Port)
	}

	return w.api
}
