package _map

import "cluster_manager/pkg/utils"

func Keys[M ~map[K]V, K comparable, V any](m M) []K {
	r := make([]K, 0, len(m))

	for k := range m {
		r = append(r, k)
	}

	return r
}

func Values[M ~map[K]V, K comparable, V any](m M) []V {
	r := make([]V, 0, len(m))

	for _, v := range m {
		r = append(r, v)
	}

	return r
}

func SumValues[V utils.Number, V2 utils.Number, M ~map[K]V2, K comparable](m M) V {
	var output V = 0

	for _, v := range m {
		output += V(v)
	}

	return output
}

func Difference[T comparable](a, b []T) []T {
	mb := make(map[T]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}

	var diff []T

	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}

	return diff
}

func ExtractField[T any](m []T, extractor func(T) string) ([]string, map[string]T) {
	var res []string

	mm := make(map[string]T)

	for i := 0; i < len(m); i++ {
		val := extractor(m[i])

		res = append(res, val)
		mm[val] = m[i]
	}

	return res, mm
}
