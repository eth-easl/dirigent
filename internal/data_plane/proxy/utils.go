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
