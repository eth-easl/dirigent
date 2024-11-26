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

package managers

import (
	"cluster_manager/api/proto"
	"cluster_manager/pkg/atomic_map"
)

type SandboxManager struct {
	Metadata atomic_map.AtomicMap[string, *Metadata]

	nodeName string
}

type RuntimeMetadata interface{}

type Metadata struct {
	ServiceName string

	RuntimeMetadata RuntimeMetadata

	HostPort  int
	IP        string
	GuestPort int
	NetNs     string

	ExitStatusChannel chan uint32
}

func NewSandboxManager(nodeName string) *SandboxManager {
	return &SandboxManager{
		Metadata: *atomic_map.NewAtomicMap[string, *Metadata](),
		nodeName: nodeName,
	}
}

func (m *SandboxManager) AddSandbox(key string, metadata *Metadata) {
	m.Metadata.Set(key, metadata)
}

func (m *SandboxManager) DeleteSandbox(key string) *Metadata {
	res, ok := m.Metadata.Get(key)
	if !ok {
		return nil
	}

	m.Metadata.Delete(key)

	return res
}

func (m *SandboxManager) ListEndpoints() (*proto.EndpointsList, error) {
	list := &proto.EndpointsList{}

	// TODO: check for consistency with containerd client list
	keys, values := m.Metadata.KeyValues()
	for i := 0; i < len(keys); i++ {
		list.Endpoint = append(list.Endpoint, &proto.Endpoint{
			SandboxID:   keys[i],
			URL:         values[i].IP,
			NodeName:    m.nodeName,
			ServiceName: values[i].ServiceName,
			HostPort:    int32(values[i].HostPort),
		})
	}

	return list, nil
}
