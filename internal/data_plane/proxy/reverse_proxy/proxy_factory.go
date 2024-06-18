package reverse_proxy

import (
	"github.com/sirupsen/logrus"
	"net"
	"net/http"
	"net/http/httputil"
	"time"
)

var EmptyReverseProxyDirector = func(request *http.Request) {}

func CreateReverseProxy() *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Director: EmptyReverseProxyDirector,
		Transport: &http.Transport{
			/*DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
			AllowHTTP:       true,
			IdleConnTimeout: 2 * time.Second,*/
			DialContext: (&net.Dialer{
				Timeout: 1 * time.Second,
			}).DialContext,
			IdleConnTimeout: 1 * time.Second,
		},
		BufferPool:    NewBufferPool(),
		FlushInterval: 0,
		ErrorHandler: func(writer http.ResponseWriter, request *http.Request, err error) {
			logrus.Errorf("Proxy error - %s - %v", request.Host, err)
		},
	}
}
