package net

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"time"

	"golang.org/x/net/http2"
	"k8s.io/apimachinery/pkg/util/wait"
)

var EmptyReverseProxyDirector = func(request *http.Request) {}

var backOffTemplate = wait.Backoff{
	Duration: 50 * time.Millisecond,
	Factor:   1.4,
	Jitter:   0.1, // At most 10% jitter.
	Steps:    15,
}

var DialWithBackOff = NewBackoffDialer(backOffTemplate)

func dialBackOffHelper(ctx context.Context, network, address string, bo wait.Backoff, tlsConf *tls.Config) (net.Conn, error) {
	dialer := &net.Dialer{
		DualStack: true,
		KeepAlive: 5 * time.Second,
		Timeout:   bo.Duration, // Initial duration.
	}
	start := time.Now()

	for {
		var (
			c   net.Conn
			err error
		)

		if tlsConf == nil {
			c, err = dialer.DialContext(ctx, network, address)
		} else {
			c, err = tls.DialWithDialer(dialer, network, address, tlsConf)
		}

		if err != nil {
			var errNet net.Error
			if errors.As(err, &errNet) && errNet.Timeout() {
				if bo.Steps < 1 {
					break
				}

				dialer.Timeout = bo.Step()

				time.Sleep(wait.Jitter(30*time.Millisecond, 1.0)) // Sleep with jitter.

				continue
			}

			return nil, err
		}

		return c, nil
	}

	elapsed := time.Since(start)

	return nil, fmt.Errorf("timed out dialing after %.2fs", elapsed.Seconds())
}

func NewBackoffDialer(backoffConfig wait.Backoff) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		return dialBackOffHelper(ctx, network, address, backoffConfig, nil)
	}
}

func NewProxy() *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Director: EmptyReverseProxyDirector,
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
				return DialWithBackOff(context.Background(), network, addr)
			},
			DisableCompression: true,
		},
		BufferPool:    NewBufferPool(),
		FlushInterval: 0,
	}
}
