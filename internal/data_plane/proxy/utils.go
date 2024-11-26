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

package proxy

import (
	common "cluster_manager/internal/data_plane/function_metadata"
	"net/http"
	"sync/atomic"
)

func GetServiceName(r *http.Request) string {
	// provided by the gRPC client through WithAuthority(...) call
	return r.Host
}

func giveBackCCCapacity(endpoint *common.UpstreamEndpoint) {
	endpointInflight := atomic.AddInt32(&endpoint.InFlight, -1)
	if atomic.LoadInt32(&endpoint.Drained) != 0 && endpointInflight == 0 {
		endpoint.DrainingCallback <- struct{}{}
	}

	if endpoint.Capacity != nil {
		endpoint.Capacity <- struct{}{}
	}
}

func contextTerminationHandler(r *http.Request, coldStartChannel chan common.ColdStartChannelStruct) {
	select {
	case <-r.Context().Done():
		if coldStartChannel != nil {
			coldStartChannel <- common.ColdStartChannelStruct{
				Outcome:             common.CanceledColdStart,
				AddEndpointDuration: 0,
			}
		}
	}
}

func HealthHandler(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(http.StatusOK)
}
