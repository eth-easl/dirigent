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
		/*{
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
		},*/
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

			metadata.addToEndpointList(endpointsInfo)
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
