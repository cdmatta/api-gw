package proxy

import (
	"net/url"
)

type Route struct {
	methods     []string
	path        string
	destination *url.URL
}

func NewRoute() *Route {
	return &Route{}
}

func (r *Route) WithMethods(methods []string) *Route {
	r.methods = methods
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
