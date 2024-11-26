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
	"cluster_manager/internal/worker_node"
	"context"
	"google.golang.org/protobuf/types/known/emptypb"
)

type WnApiServer struct {
	proto.UnimplementedWorkerNodeInterfaceServer

	workerNode *worker_node.WorkerNode
}

func NewWorkerNodeApi(workerNode *worker_node.WorkerNode) *WnApiServer {
	return &WnApiServer{
		workerNode: workerNode,
	}
}

func (w *WnApiServer) CreateSandbox(grpcCtx context.Context, in *proto.ServiceInfo) (*proto.SandboxCreationStatus, error) {
	return w.workerNode.SandboxRuntime.CreateSandbox(grpcCtx, in)
}

func (w *WnApiServer) DeleteSandbox(grpcCtx context.Context, in *proto.SandboxID) (*proto.ActionStatus, error) {
	return w.workerNode.SandboxRuntime.DeleteSandbox(grpcCtx, in)
}

func (w *WnApiServer) ListEndpoints(grpcCtx context.Context, in *emptypb.Empty) (*proto.EndpointsList, error) {
	return w.workerNode.SandboxRuntime.ListEndpoints(grpcCtx, in)
}
