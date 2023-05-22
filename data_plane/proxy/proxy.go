package proxy

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"k8s.io/apimachinery/pkg/util/wait"
	"net"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"
)

const sleep = 30 * time.Millisecond

func dialBackOffHelper(ctx context.Context, network, address string, bo wait.Backoff, tlsConf *tls.Config) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout:   bo.Duration, // Initial duration.
		KeepAlive: 5 * time.Second,
		DualStack: true,
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
				time.Sleep(wait.Jitter(sleep, 1.0)) // Sleep with jitter.
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

var backOffTemplate = wait.Backoff{
	Duration: 50 * time.Millisecond,
	Factor:   1.4,
	Jitter:   0.1, // At most 10% jitter.
	Steps:    15,
}

var DialWithBackOff = NewBackoffDialer(backOffTemplate)

type bufferPool struct {
	pool *sync.Pool
}

// NewBufferPool creates a new BufferPool. This is only safe to use in the context
// of a httputil.ReverseProxy, as the buffers returned via Put are not cleaned
// explicitly.
func NewBufferPool() httputil.BufferPool {
	return &bufferPool{
		// We don't use the New function of sync.Pool here to avoid an unnecessary
		// allocation when creating the slices. They are implicitly created in the
		// Get function below.
		pool: &sync.Pool{},
	}
}

func (b *bufferPool) Get() []byte {
	buf := b.pool.Get()
	if buf == nil {
		// Use the default buffer size as defined in the ReverseProxy itself.
		return make([]byte, 32*1024)
	}

	return *buf.(*[]byte)
}

// Put returns the given Buffer to the bufferPool.
func (b *bufferPool) Put(buffer []byte) {
	b.pool.Put(&buffer)
}

func ProxyHandler(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for k, v := range r.Header {
			fmt.Printf("%s %s\n", k, v)
		}

		next.ServeHTTP(w, r)
	}
}
