package proxy

import (
	"cluster_manager/internal/data_plane/service_metadata"
	"context"
	"net/http"
	"sync/atomic"
)

func GetServiceName(r *http.Request) string {
	// provided by the gRPC client through WithAuthority(...) call
	return r.Host
}

func giveBackCCCapacity(endpoint *service_metadata.UpstreamEndpoint) {
	endpointInflight := atomic.AddInt32(&endpoint.InFlight, -1)
	if atomic.LoadInt32(&endpoint.Drained) != 0 && endpointInflight == 0 {
		endpoint.DrainingCallback <- struct{}{}
	}

	if endpoint.Capacity != nil {
		endpoint.Capacity <- struct{}{}
	}
}

func contextTerminationHandler(c context.Context, coldStartChannel chan service_metadata.ColdStartChannelStruct) {
	select {
	case <-c.Done():
		if coldStartChannel != nil {
			coldStartChannel <- service_metadata.ColdStartChannelStruct{
				Outcome:             service_metadata.CanceledColdStart,
				AddEndpointDuration: 0,
			}
		}
	}
}

func HealthHandler(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(http.StatusOK)
}
