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

package api

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/data_plane"
	"cluster_manager/internal/data_plane/proxy"
	"context"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
)

type DpApiServer struct {
	proto.UnimplementedDpiInterfaceServer

	dataplane *data_plane.Dataplane
	Proxy     *proxy.ProxyingService
}

func NewDpApiServer(dataPlane *data_plane.Dataplane) *DpApiServer {
	return &DpApiServer{
		dataplane: dataPlane,
	}
}

func (api *DpApiServer) AddDeployment(_ context.Context, in *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error) {
	return api.dataplane.AddDeployment(in)
}

func (api *DpApiServer) UpdateEndpointList(_ context.Context, patch *proto.DeploymentEndpointPatch) (*proto.DeploymentUpdateSuccess, error) {
	return api.dataplane.UpdateEndpointList(patch)
}

func (api *DpApiServer) DeleteDeployment(_ context.Context, name *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error) {
	return api.dataplane.DeleteDeployment(name)
}

func (api *DpApiServer) DrainSandbox(_ context.Context, endpoint *proto.DeploymentEndpointPatch) (*proto.DeploymentUpdateSuccess, error) {
	return api.dataplane.DrainSandbox(endpoint)
}

// TODO: Remove this function
func (api *DpApiServer) ResetMeasurements(_ context.Context, in *emptypb.Empty) (*proto.ActionStatus, error) {
	logrus.Warn("This function does nothing")
	return &proto.ActionStatus{Success: true}, nil
}
