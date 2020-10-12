package middleware

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

type AccessLoggingMetricsMiddleware struct{}

var gatewayRequestsDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{Name: "gateway_requests_seconds"},
	[]string{"method", "status", "uri"},
)

func NewAccessLoggingMetricsMiddleware() *AccessLoggingMetricsMiddleware {
	return &AccessLoggingMetricsMiddleware{}
}

func (a *AccessLoggingMetricsMiddleware) getPriority() int {
	return PriorityAccessLoggingMetricsMiddleware
}

func (a *AccessLoggingMetricsMiddleware) FilterFunction(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		remoteAddress := r.RemoteAddr
		method := r.Method
		uri := r.RequestURI // TODO replace with route path template ?
		protocol := r.Proto
		referer := r.Referer()
		userAgent := r.UserAgent()
		lrw := newLoggingResponseWriter(w)
		statusCode := http.StatusOK
		timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
			gatewayRequestsDuration.WithLabelValues(method, strconv.Itoa(statusCode), uri).Observe(v)
		}))

		next.ServeHTTP(lrw, r)

		statusCode = lrw.statusCode
		duration := timer.ObserveDuration().Milliseconds()
		zap.S().Infof("%s %s %s %s %d '%s' '%s' %d", remoteAddress, method, uri, protocol, statusCode, referer, userAgent, duration)
	}
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

func (l *loggingResponseWriter) WriteHeader(code int) {
	l.statusCode = code
	l.ResponseWriter.WriteHeader(code)
}
