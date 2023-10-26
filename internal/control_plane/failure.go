package control_plane

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/core"
	"cluster_manager/pkg/synchronization"
	"github.com/sirupsen/logrus"
	"sync/atomic"
)

func (c *ControlPlane) HandleFailure(failures []*proto.Failure) bool {
	for _, failure := range failures {
		switch failure.Type {
		case proto.FailureType_SANDBOX_FAILURE, proto.FailureType_WORKER_NODE_FAILURE:
			// RECOVERY ACTION: reduce the actual scale by one and propagate endpoint removal
			serviceName := failure.ServiceName
			sandboxIDs := failure.SandboxIDs

			// TODO: all endpoints removals can be batched into a single call
			if ss, ok := c.SIStorage.Get(serviceName); ok {
				c.removeEndpoints(ss, c.DataPlaneConnections, sandboxIDs)
			}
		}
	}

	return true
}

func (c *ControlPlane) removeEndpoints(ss *ServiceInfoStorage, dpConns synchronization.SyncStructure[string, core.DataPlaneInterface], endpoints []string) {
	ss.Controller.EndpointLock.Lock()
	defer ss.Controller.EndpointLock.Unlock()

	toRemove := make(map[*core.Endpoint]struct{})
	for _, cid := range endpoints {
		endpoint := searchEndpointByContainerName(ss.Controller.Endpoints, cid)
		if endpoint == nil {
			logrus.Warn("Endpoint ", cid, " not found for removal.")
			continue
		}

		toRemove[endpoint] = struct{}{}
		ss.removeEndpointFromWNStruct(endpoint)

		logrus.Warn("Control plane notified of failure of '", endpoint.SandboxID, "'. Decrementing actual scale and removing the endpoint.")
	}

	atomic.AddInt64(&ss.Controller.ScalingMetadata.ActualScale, -int64(len(toRemove)))
	ss.excludeEndpoints(toRemove)

	ss.updateEndpoints(dpConns, ss.prepareUrlList())
}
