package proxy

import (
	"cluster_manager/internal/data_plane/function_metadata"
	"encoding/json"
	"net/http"
)

func CreateMetricsHandler(deployments *function_metadata.Deployments) func(writer http.ResponseWriter, r *http.Request) {
	return func(writer http.ResponseWriter, r *http.Request) {
		var statistics *function_metadata.FunctionStatistics

		service := r.URL.Query().Get("service")
		if service == "" {
			statistics = function_metadata.AggregateStatistics(deployments.ListFunctions())
		} else {
			metadata, _ := deployments.GetFunctionDeployment(service)
			if metadata == nil {
				writer.WriteHeader(http.StatusBadRequest)
				return
			}

			statistics = metadata.GetStatistics()
		}

		data, err := json.Marshal(*statistics)
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, _ = writer.Write(data)
		writer.WriteHeader(http.StatusOK)
	}
}
