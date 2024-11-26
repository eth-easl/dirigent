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

package eviction_policy

import (
	"cluster_manager/internal/control_plane/core"
	"cluster_manager/pkg/tracing"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEvictionPolicyEmpty(t *testing.T) {
	emptyList := make([]*core.Endpoint, 0)

	toEvict, currentState := EvictionPolicy(emptyList)

	assert.Nil(t, toEvict, "Evicted point should be nil")
	assert.Len(t, currentState, 0, "No object should ne present in the list")
}

func TestEvictionPolicy(t *testing.T) {
	list := make([]*core.Endpoint, 0)

	list = append(list, &core.Endpoint{
		SandboxID:       "",
		URL:             "",
		Node:            nil,
		HostPort:        0,
		CreationHistory: tracing.ColdStartLogEntry{},
	})

	list = append(list, &core.Endpoint{
		SandboxID:       "",
		URL:             "",
		Node:            nil,
		HostPort:        0,
		CreationHistory: tracing.ColdStartLogEntry{},
	})

	list = append(list, &core.Endpoint{
		SandboxID:       "",
		URL:             "",
		Node:            nil,
		HostPort:        0,
		CreationHistory: tracing.ColdStartLogEntry{},
	})

	toEvict, currentState := EvictionPolicy(list)

	assert.NotNil(t, toEvict, "Evicted point should not be nil")
	assert.Len(t, currentState, 2, "Two objects should be present in the list")
}
