package control_plane

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/internal/control_plane/control_plane/endpoint_placer"
	"cluster_manager/proto"
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
				c.removeEndpoints(ss, sandboxIDs)
			}
		}
	}

	return true
}

func (c *ControlPlane) removeEndpoints(ss *endpoint_placer.EndpointPlacer, endpoints []string) {
	ss.PerFunctionState.EndpointLock.Lock()
	defer ss.PerFunctionState.EndpointLock.Unlock()

	toRemove := make(map[*core.Endpoint]struct{})
	for _, cid := range endpoints {
		endpoint := searchEndpointByContainerName(ss.PerFunctionState.Endpoints, cid)
		if endpoint == nil {
			logrus.Warn("Endpoint ", cid, " not found for removal.")
			continue
		}

		toRemove[endpoint] = struct{}{}
		ss.RemoveEndpointFromWNStruct(endpoint)

		logrus.Warn("Control plane notified of failure of '", endpoint.SandboxID, "'. Decrementing actual scale and removing the endpoint.")
	}

	atomic.AddInt64(&ss.PerFunctionState.ActualScale, -int64(len(toRemove)))
	ss.ExcludeEndpoints(toRemove)

	ss.UpdateEndpoints(ss.PrepareEndpointInfo(ss.PerFunctionState.Endpoints))
}
