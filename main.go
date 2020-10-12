package main

import (
	"os"

	"github.com/cdmatta/api-gw/config"
	"github.com/cdmatta/api-gw/proxy"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	logger := initZapLog()
	defer logger.Sync()

	if len(os.Args) == 1 {
		zap.S().Fatalf("usage: %s <config-file>", os.Args[0])
	}

	configFile := os.Args[1]
	apiGwConfig, err := config.LoadConfig(configFile)
	if err != nil {
		zap.S().Fatal(err)
	}
	zap.S().Infof("%+v", apiGwConfig)

	gw := proxy.NewReverseProxy()
	for _, routeConfig := range apiGwConfig.Routes {
		url, err := routeConfig.BackendConfig.GetUrl()
		if err != nil {
			zap.S().Fatal(err)
		}

		r := proxy.NewRoute().
			WithMethods(routeConfig.Methods).
			WithPath(routeConfig.Path).
			WithDestination(url)

		gw.SetRoute(r)
	}

	zap.S().Infof("Starting gateway on %s", apiGwConfig.Server.GetListenAddress())
	gw.ListenAndServe(apiGwConfig.Server.GetListenAddress())
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
