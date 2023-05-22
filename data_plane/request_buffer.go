package main

import (
	proxy2 "cluster_manager/proxy"
	"context"
	"crypto/tls"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"net"
	"net/http"
	"net/http/httputil"
)

func main() {
	proxy := &httputil.ReverseProxy{
		Director: func(request *http.Request) {
			target := "localhost:80"
			request.URL.Scheme = "http"
			request.URL.Host = target
		},
	}
	proxy.Transport = &http2.Transport{
		DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
			return proxy2.DialWithBackOff(context.Background(), network, addr)
		},
		DialTLS: func(netw, addr string, _ *tls.Config) (net.Conn, error) {
			return proxy2.DialWithBackOff(context.Background(), netw, addr)
		},
		DisableCompression: true,
		AllowHTTP:          true,
	}
	proxy.BufferPool = proxy2.NewBufferPool()
	proxy.FlushInterval = 0

	var composedHandler http.Handler = proxy
	composedHandler = proxy2.ProxyHandler(composedHandler)
	composedHandler = proxy2.ForwardedShimHandler(composedHandler)

	server := &http.Server{
		Addr:    "localhost:8080",
		Handler: h2c.NewHandler(composedHandler, &http2.Server{}),
	}

	server.ListenAndServe()
}
