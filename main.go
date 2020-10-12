package main

import (
	"net/url"

	"github.com/cdmatta/api-gw/proxy"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	logger := initZapLog()
	defer logger.Sync()

	gw := proxy.NewReverseProxy()

	backend, _ := url.Parse("http://127.0.0.1:8080")
	r := proxy.NewRoute().
		WithMethod("GET").
		WithPath("/hw").
		WithDestination(backend)
	gw.SetRoute(r)

	zap.S().Infof("Starting gateway on %s", ":8080")
	gw.ListenAndServe(":8080")
}

func initZapLog() *zap.Logger {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.CallerKey = ""
	logger, _ := cfg.Build()
	zap.ReplaceGlobals(logger)
	return logger
}
