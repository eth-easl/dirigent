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
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

const (
	mockHost string = "mockHost"
	mockkey  string = "mockKey"
)

func TestSimpleAddDelete(t *testing.T) {
	manager := NewSandboxManager(mockHost)

	manager.AddSandbox(mockkey, &Metadata{})
	manager.DeleteSandbox(mockkey)
}

func TestSandboxManager(t *testing.T) {
	i := 0
	manager := NewSandboxManager(mockHost)

	for ; i <= 1000; i++ {
		manager.AddSandbox(strconv.Itoa(i), &Metadata{})
		assert.Equal(t, manager.Metadata.Len(), i+1)
	}

	i--

	for ; i >= 0; i-- {
		manager.DeleteSandbox(strconv.Itoa(i))
		assert.Equal(t, manager.Metadata.Len(), i)
	}
}
