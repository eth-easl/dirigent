package load_balancing

import (
	"cluster_manager/internal/common"
	"testing"

	"github.com/stretchr/testify/assert"
)

func getTestEndpoints() []*common.UpstreamEndpoint {
	return []*common.UpstreamEndpoint{
		{
			ID: "1",
		},
		{
			ID: "2",
		},
		{
			ID: "3",
		},
		{
			ID: "4",
		},
		{
			ID: "5",
		},
		{
			ID: "6",
		},
		{
			ID: "7",
		},
	}
}

func TestRandomLoadBalancing(t *testing.T) {
	endpoints := getTestEndpoints()

	endpointsMap := make(map[*common.UpstreamEndpoint]interface{})
	for _, elem := range endpoints {
		endpointsMap[elem] = 0
	}

	for i := 0; i < 10; i++ {
		endpoint := randomLoadBalancing(endpoints)
		if _, ok := endpointsMap[endpoint]; ok {
			assert.True(t, ok, "Element isn't present in the endpoints")
		}
	}
}

func TestRoundRobinLoadBalancing(t *testing.T) {
	endpoints := getTestEndpoints()

	for i := 0; i < 100; i++ {
		for j := 0; j < len(endpoints); j++ {
			endpoint := roundRobinLoadBalancing(endpoints)
			assert.Equal(t, endpoints[j], endpoint, "Endpoints isn't the correct one")
		}
	}
}
