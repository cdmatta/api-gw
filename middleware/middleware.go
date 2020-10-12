package middleware

import (
	"net/http"
	"sort"
)

type FilterFunctionAdaptor func(http.HandlerFunc) http.HandlerFunc

type Middleware interface {
	getPriority() int
	FilterFunction(http.HandlerFunc) http.HandlerFunc
}

func Compose(middlewares ...Middleware) FilterFunctionAdaptor {
	sort.Slice(middlewares, func(i, j int) bool {
		return middlewares[i].getPriority() < middlewares[j].getPriority()
	})

	return func(next http.HandlerFunc) http.HandlerFunc {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i].FilterFunction(next)
		}
		return next
	}
}

const (
	PriorityAccessLoggingMetricsMiddleware = iota
)
