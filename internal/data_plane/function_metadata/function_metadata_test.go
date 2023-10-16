package function_metadata

import (
	"cluster_manager/api/proto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEndpointMerge(t *testing.T) {
	tests := []struct {
		testName     string
		oldURLs      []string
		newURLs      []string
		expectedURLs []string
	}{
		{
			testName:     "endpoint_merge_1nil_2new",
			oldURLs:      nil,
			newURLs:      []string{"a", "b", "c"},
			expectedURLs: []string{"a", "b", "c"},
		},
		{
			testName:     "endpoint_merge_1empty_2new",
			oldURLs:      []string{},
			newURLs:      []string{"a", "b", "c"},
			expectedURLs: []string{"a", "b", "c"},
		},
		{
			testName:     "endpoint_merge_1two_2append_one",
			oldURLs:      []string{"a", "b"},
			newURLs:      []string{"a", "b", "c"},
			expectedURLs: []string{"a", "b", "c"},
		},
		{
			testName:     "endpoint_merge_1two_2delete_one",
			oldURLs:      []string{"a", "b"},
			newURLs:      []string{"b", "c"},
			expectedURLs: []string{"b", "c"},
		},
		{
			testName:     "endpoint_merge_1two_2add_one_delete_one",
			oldURLs:      []string{"a", "b"},
			newURLs:      []string{"b", "c", "d"},
			expectedURLs: []string{"b", "c", "d"},
		},
		{
			testName:     "endpoint_merge_1two_2change_completely",
			oldURLs:      []string{"a", "b"},
			newURLs:      []string{"c", "d", "e"},
			expectedURLs: []string{"c", "d", "e"},
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

			metadata.updateEndpointList(endpointsInfo)
			mergedResults := metadata.upstreamEndpoints

			assert.Equal(t, len(mergedResults), len(test.expectedURLs), "Invalid endpoint merge. Algorithm is broken.")

			for i := 0; i < len(mergedResults); i++ {
				found := false
				url1 := mergedResults[i]

				for j := 0; j < len(test.expectedURLs); j++ {
					url2 := mergedResults[j]

					if url1 == url2 {
						found = true
						break
					}
				}

				assert.True(t, found, "Invalid endpoint merge. Algorithm is broken.")
			}
		})
	}
}
