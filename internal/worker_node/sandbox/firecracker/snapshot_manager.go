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

package firecracker

import (
	"cluster_manager/internal/worker_node/managers"
)

type SnapshotManager struct {
	managers.RootFsManager[*SnapshotMetadata]
}

type SnapshotMetadata struct {
	MemoryPath   string
	SnapshotPath string
}

func NewFirecrackerSnapshotManager() *SnapshotManager {
	return &SnapshotManager{}
}

func (sm *SnapshotManager) Exists(serviceName string) bool {
	_, ok := sm.Get(serviceName)

	return ok
}

func (sm *SnapshotManager) AddSnapshot(serviceID string, metadata *SnapshotMetadata) {
	sm.LoadOrStore(serviceID, metadata)
}

func (sm *SnapshotManager) FindSnapshot(serviceName string) (*SnapshotMetadata, bool) {
	return sm.Get(serviceName)
}
