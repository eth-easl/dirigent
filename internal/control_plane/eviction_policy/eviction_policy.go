package eviction_policy

import (
	"cluster_manager/internal/control_plane/core"
	"math/rand"
)

func EvictionPolicy(endpoint []*core.Endpoint) (*core.Endpoint, []*core.Endpoint) {
	if len(endpoint) == 0 {
		return nil, []*core.Endpoint{}
	}

	index := rand.Intn(len(endpoint))

	var newEndpointList []*core.Endpoint

	for i := 0; i < len(endpoint); i++ {
		if i != index {
			newEndpointList = append(newEndpointList, endpoint[i])
		}
	}

	return endpoint[index], newEndpointList
}
