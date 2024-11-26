/*
 * MIT License
 *
 * Copyright (c) 2024 EASL
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

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

	ss.updateEndpoints(ss.prepareEndpointInfo(ss.Controller.Endpoints))
}
