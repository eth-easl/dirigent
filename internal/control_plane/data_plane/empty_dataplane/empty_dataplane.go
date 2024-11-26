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

package empty_dataplane

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/core"
	"context"
	"google.golang.org/protobuf/types/known/emptypb"
	"time"
)

type emptyDataplane struct{}

func NewDataplaneConnectionEmpty(IP, APIPort, ProxyPort string) core.DataPlaneInterface {
	return &emptyDataplane{}
}

func (e emptyDataplane) InitializeDataPlaneConnection(host string, port string) error {
	return nil
}

func (e emptyDataplane) AddDeployment(ctx context.Context, info *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error) {
	return &proto.DeploymentUpdateSuccess{
		Success: true,
		Message: "",
	}, nil
}

func (e emptyDataplane) UpdateEndpointList(ctx context.Context, patch *proto.DeploymentEndpointPatch) (*proto.DeploymentUpdateSuccess, error) {
	return &proto.DeploymentUpdateSuccess{
		Success: true,
		Message: "",
	}, nil
}

func (e emptyDataplane) DeleteDeployment(ctx context.Context, info *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error) {
	return &proto.DeploymentUpdateSuccess{
		Success: true,
		Message: "",
	}, nil
}

func (e emptyDataplane) DrainSandbox(context.Context, *proto.DeploymentEndpointPatch) (*proto.DeploymentUpdateSuccess, error) {
	return &proto.DeploymentUpdateSuccess{
		Success: true,
		Message: "",
	}, nil
}

func (e emptyDataplane) ResetMeasurements(ctx context.Context, empty *emptypb.Empty) (*proto.ActionStatus, error) {
	return &proto.ActionStatus{
		Success: true,
		Message: "",
	}, nil
}

func (e emptyDataplane) GetLastHeartBeat() time.Time {
	return time.Now()
}

func (e emptyDataplane) UpdateHeartBeat() {
}

func (e emptyDataplane) GetIP() string {
	return ""
}

func (e emptyDataplane) GetApiPort() string {
	return ""
}

func (e emptyDataplane) GetProxyPort() string {
	return ""
}
