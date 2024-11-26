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

package fake_snapshot

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/worker_node/sandbox"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"math/rand"
	"time"
)

type Runtime struct {
	sandbox.RuntimeInterface
}

func NewFakeSnapshotRuntime() *Runtime {
	return &Runtime{}
}

func (fsr *Runtime) CreateSandbox(_ context.Context, in *proto.ServiceInfo) (*proto.SandboxCreationStatus, error) {
	time.Sleep(40 * time.Millisecond)
	logrus.Debugf("Fake sandbox created successfully.")

	zeroDuration := durationpb.New(0)

	return &proto.SandboxCreationStatus{
		Success: true,
		ID:      fmt.Sprintf("dummy-sandbox-%d", rand.Int()),
		PortMappings: &proto.PortMapping{
			HostPort:  80,
			GuestPort: 80,
			Protocol:  proto.L4Protocol_TCP,
		},
		LatencyBreakdown: &proto.SandboxCreationBreakdown{
			Total:                zeroDuration,
			ImageFetch:           zeroDuration,
			SandboxCreate:        zeroDuration,
			NetworkSetup:         zeroDuration,
			SandboxStart:         zeroDuration,
			Iptables:             zeroDuration,
			ReadinessProbing:     zeroDuration,
			DataplanePropagation: zeroDuration,
			SnapshotCreation:     zeroDuration,
			ConfigureMonitoring:  zeroDuration,
			FindSnapshot:         zeroDuration,
		},
	}, nil
}

func (fsr *Runtime) DeleteSandbox(_ context.Context, _ *proto.SandboxID) (*proto.ActionStatus, error) {
	return &proto.ActionStatus{Success: true}, nil
}

func (fsr *Runtime) ListEndpoints(_ context.Context, _ *emptypb.Empty) (*proto.EndpointsList, error) {
	return &proto.EndpointsList{Endpoint: nil}, nil
}

func (fsr *Runtime) ValidateHostConfig() bool {
	return true
}
