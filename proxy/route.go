package proxy

import (
	"net/url"
)

type Route struct {
	method      string
	path        string
	destination *url.URL
}

func NewRoute() *Route {
	return &Route{}
}

func (r *Route) WithMethod(method string) *Route {
	r.method = method
	return r
}

func (r *Route) WithPath(path string) *Route {
	r.path = path
	return r
}

func (r *Route) WithDestination(destination *url.URL) *Route {
	r.destination = destination
	return r
}
