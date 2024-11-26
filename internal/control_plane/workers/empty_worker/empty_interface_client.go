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

package empty_worker

import (
	"cluster_manager/api/proto"
	"context"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type emptyInterfaceClient struct {
}

func NewEmptyInterfaceClient() proto.WorkerNodeInterfaceClient {
	return &emptyInterfaceClient{}
}

func (e emptyInterfaceClient) CreateSandbox(ctx context.Context, in *proto.ServiceInfo, opts ...grpc.CallOption) (*proto.SandboxCreationStatus, error) {
	return &proto.SandboxCreationStatus{
		Success: true,
	}, nil
}

func (e emptyInterfaceClient) DeleteSandbox(ctx context.Context, in *proto.SandboxID, opts ...grpc.CallOption) (*proto.ActionStatus, error) {
	return &proto.ActionStatus{
		Success: true,
	}, nil
}

func (e emptyInterfaceClient) ListEndpoints(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*proto.EndpointsList, error) {
	return &proto.EndpointsList{}, nil
}
