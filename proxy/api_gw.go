package proxy

import (
	"net/http"
	"net/http/httputil"

	"github.com/julienschmidt/httprouter"
)

type ReverseProxy struct {
	router httprouter.Router
}

func NewReverseProxy() *ReverseProxy {
	return &ReverseProxy{}
}

func (r *ReverseProxy) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, &r.router)
}

func (r *ReverseProxy) SetRoute(route *Route) {
	r.router.Handler(route.method, route.path, newReverseProxyHandler(route))
}

func newReverseProxyHandler(route *Route) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			dst := route.destination

			req.Host = dst.Host
			req.URL.Scheme = dst.Scheme
			req.URL.Host = dst.Host
			req.URL.Path = dst.Path

			req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		},
	}
}
