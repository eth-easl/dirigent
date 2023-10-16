package function_metadata

import (
	"cluster_manager/api/proto"
	"testing"
)

func TestEndpointMerge(t *testing.T) {
	tests := []struct {
		testName     string
		oldURLs      []string
		newURLs      []string
		toProbe      []string
		expectedURLs []string
	}{
		{
			testName:     "endpoint_merge_1nil_2new",
			oldURLs:      nil,
			newURLs:      []string{"a", "b", "c"},
			toProbe:      []string{"a", "b", "c"},
			expectedURLs: []string{},
		},
		{
			testName:     "endpoint_merge_1empty_2new",
			oldURLs:      []string{},
			newURLs:      []string{"a", "b", "c"},
			toProbe:      []string{"a", "b", "c"},
			expectedURLs: []string{},
		},
		{
			testName:     "endpoint_merge_1two_2append_one",
			oldURLs:      []string{"a", "b"},
			newURLs:      []string{"a", "b", "c"},
			toProbe:      []string{"c"},
			expectedURLs: []string{"a", "b"},
		},
		{
			testName:     "endpoint_merge_1two_2delete_one",
			oldURLs:      []string{"a", "b"},
			newURLs:      []string{"b", "c"},
			toProbe:      []string{"c"},
			expectedURLs: []string{"b"},
		},
		{
			testName:     "endpoint_merge_1two_2add_one_delete_one",
			oldURLs:      []string{"a", "b"},
			newURLs:      []string{"b", "c", "d"},
			toProbe:      []string{"c", "d"},
			expectedURLs: []string{"b"},
		},
		{
			testName:     "endpoint_merge_1two_2change_completely",
			oldURLs:      []string{"a", "b"},
			newURLs:      []string{"c", "d", "e"},
			toProbe:      []string{"c", "d", "e"},
			expectedURLs: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			var endpoints []*UpstreamEndpoint
			for i := 0; i < len(test.oldURLs); i++ {
				endpoints = append(endpoints, &UpstreamEndpoint{
					URL: test.oldURLs[i],
				})
			}

			metadata := FunctionMetadata{
				identifier:        test.testName,
				upstreamEndpoints: endpoints,
			}

			endpointsInfo := make([]*proto.EndpointInfo, 0)
			for _, elem := range test.newURLs {
				endpointsInfo = append(endpointsInfo, &proto.EndpointInfo{
					ID:  "mock_id",
					URL: elem,
				})
			}

			toProbe := metadata.mergeEndpointLists(endpointsInfo)

			// Endpoint adding
			if len(toProbe) != len(test.toProbe) {
				t.Error("Invalid merge endpoint list function return value.")
			}

			for i := 0; i < len(test.toProbe); i++ {
				if test.toProbe[i] != toProbe[i].URL {
					t.Error("Invalid endpoint to probe merge. Algorithm is broken.")
				}
			}

			// Endpoint removal
			if len(metadata.upstreamEndpoints) != len(test.expectedURLs) {
				t.Error("Invalid endpoint list function return value.")
			}

			for i := 0; i < len(test.expectedURLs); i++ {
				if test.expectedURLs[i] != metadata.upstreamEndpoints[i].URL {
					t.Error("Invalid endpoint merge. Algorithm is broken.")
				}
			}
		})
	}
}
