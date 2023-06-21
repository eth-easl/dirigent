package common

import (
	"testing"
)

func TestDifference(t *testing.T) {
	s1 := []string{"a", "b", "c"}
	s2 := []string{"b", "c", "d"}

	lr := difference(s1, s2)
	if lr == nil || len(lr) != 1 || lr[0] != "a" {
		t.Error("s1 \\ s2 yielded a wrong result.")
	}

	rr := difference(s2, s1)
	if rr == nil || len(rr) != 1 || rr[0] != "d" {
		t.Error("s2 \\ s1 yielded a wrong result.")
	}
}

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
			var endpoints []UpstreamEndpoint
			for i := 0; i < len(test.oldURLs); i++ {
				endpoints = append(endpoints, UpstreamEndpoint{
					URL: test.oldURLs[i],
				})
			}

			metadata := FunctionMetadata{
				identifier:        test.testName,
				upstreamEndpoints: endpoints,
			}

			metadata.mergeEndpointList(test.newURLs)

			mergedResults := metadata.getAllUrls()

			if len(mergedResults) != len(test.expectedURLs) {
				t.Error("Invalid endpoint merge. Algorithm is broken.")
			}

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

				if !found {
					t.Error("Invalid endpoint merge. Algorithm is broken.")
				}
			}
		})
	}
}
