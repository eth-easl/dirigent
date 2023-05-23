package proxy

import (
	"net/http"
)

func LoadBalancingHandler(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: place to add load balancer

		target := "localhost:80"
		r.URL.Scheme = "http"
		r.URL.Host = target

		next.ServeHTTP(w, r)
	}
}
