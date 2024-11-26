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

package containerd

import (
	"cluster_manager/api/proto"
	"math/rand"
	"sync"
)

var portsInUse = make(map[int]struct{})
var portsInUseMutex = sync.Mutex{}

func l4ProtocolToString(protocol proto.L4Protocol) string {
	switch protocol.Number() {
	case 1:
		return "udp"
	default:
		return "tcp"
	}
}

func validatePort(port int32) bool {
	if port <= 0 || port >= 65536 {
		return false
	}

	return true
}

func AssignRandomPort() int {
	portsInUseMutex.Lock()
	defer portsInUseMutex.Unlock()

	var port int

	for {
		port = 1024 + rand.Intn(64000)

		_, ok := portsInUse[port]
		if !ok {
			portsInUse[port] = struct{}{}

			break
		}
	}

	return port
}

func UnassignPort(port int) {
	portsInUseMutex.Lock()
	defer portsInUseMutex.Unlock()

	delete(portsInUse, port)
}
