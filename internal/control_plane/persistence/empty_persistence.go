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

package persistence

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/core"
	"context"
)

func NewEmptyPeristenceLayer() *EmptyPersistence {
	return &EmptyPersistence{}
}

type EmptyPersistence struct{}

func (e *EmptyPersistence) StoreDataPlaneInformation(_ context.Context, _ *proto.DataplaneInformation) error {
	return nil
}

func (e *EmptyPersistence) DeleteDataPlaneInformation(_ context.Context, _ *proto.DataplaneInformation) error {
	return nil
}

func (e *EmptyPersistence) GetDataPlaneInformation(_ context.Context) ([]*proto.DataplaneInformation, error) {
	return make([]*proto.DataplaneInformation, 0), nil
}

func (e *EmptyPersistence) StoreWorkerNodeInformation(_ context.Context, _ *proto.WorkerNodeInformation) error {
	return nil
}

func (e *EmptyPersistence) DeleteWorkerNodeInformation(_ context.Context, _ string) error {
	return nil
}

func (e *EmptyPersistence) GetWorkerNodeInformation(_ context.Context) ([]*proto.WorkerNodeInformation, error) {
	return make([]*proto.WorkerNodeInformation, 0), nil
}

func (e *EmptyPersistence) StoreServiceInformation(_ context.Context, _ *proto.ServiceInfo) error {
	return nil
}

func (e *EmptyPersistence) DeleteServiceInformation(_ context.Context, _ *proto.ServiceInfo) error {
	return nil
}

func (e *EmptyPersistence) GetServiceInformation(_ context.Context) ([]*proto.ServiceInfo, error) {
	return make([]*proto.ServiceInfo, 0), nil
}

func (e *EmptyPersistence) SetLeader(_ context.Context) error {
	return nil
}

func (e *EmptyPersistence) StoreEndpoint(ctx context.Context, endpoint core.Endpoint) error {
	return nil
}
