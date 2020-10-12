package proxy

import (
	"net/http"
	"net/http/httputil"

	"github.com/cdmatta/api-gw/middleware"
	"github.com/julienschmidt/httprouter"
)

type ReverseProxy struct {
	router           httprouter.Router
	globalFilterFunc http.HandlerFunc
}

func NewReverseProxy() *ReverseProxy {
	return &ReverseProxy{}
}

func (r *ReverseProxy) WithGlobalFilterFunc(m middleware.FilterFunctionAdaptor) *ReverseProxy {
	r.globalFilterFunc = m(func(w http.ResponseWriter, req *http.Request) {
		r.router.ServeHTTP(w, req)
	})
	return r
}

func (r *ReverseProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.globalFilterFunc(w, req)
}

func (r *ReverseProxy) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, r)
}

func (r *ReverseProxy) SetRoute(route *Route) {
	for _, method := range route.methods {
		r.router.Handler(method, route.path, newReverseProxyHandler(route))
	}
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
