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
			DialContext: (&net.Dialer{
				Timeout: 10 * time.Second,
			}).DialContext,
			DisableCompression:  true,
			IdleConnTimeout:     60 * time.Second,
			MaxIdleConns:        3000,
			MaxIdleConnsPerHost: 3000,
		},
		BufferPool:    NewBufferPool(),
		FlushInterval: 0,
		ErrorHandler: func(writer http.ResponseWriter, request *http.Request, err error) {
			logrus.Errorf("Proxy error - %s - %v", request.Host, err)
		},
	}
}
