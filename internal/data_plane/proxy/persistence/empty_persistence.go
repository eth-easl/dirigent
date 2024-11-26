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

package request_persistence

import (
	"cluster_manager/internal/data_plane/proxy/requests"
	"context"
	"time"
)

type emptyRequestPersistence struct {
}

func CreateEmptyRequestPersistence() RequestPersistence {
	return &emptyRequestPersistence{}
}

func (empty *emptyRequestPersistence) PersistBufferedRequest(ctx context.Context, request *requests.BufferedRequest) (error, time.Duration, time.Duration) {
	return nil, 0, 0
}

func (empty *emptyRequestPersistence) PersistBufferedResponse(ctx context.Context, response *requests.BufferedResponse) error {
	return nil
}

func (empty *emptyRequestPersistence) ScanBufferedRequests(ctx context.Context) ([]*requests.BufferedRequest, error) {
	return make([]*requests.BufferedRequest, 0), nil
}

func (empty *emptyRequestPersistence) ScanBufferedResponses(ctx context.Context) ([]*requests.BufferedResponse, error) {
	return make([]*requests.BufferedResponse, 0), nil
}

func (empty *emptyRequestPersistence) DeleteBufferedRequest(ctx context.Context, code string) error {
	return nil
}
