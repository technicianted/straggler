package cmd

import (
	"net/http"

	"stagger/pkg/metrics"
	"stagger/pkg/version"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var RootCMD = &cobra.Command{
	Use:   "aicon",
	Short: "openai service router",
}

var (
	LogVerbosity           int
	ProductionStyleLogging bool
	MetricsListenAddress   string
	PProfListenAddress     string
)

var (
	// BuildInfo is a metric that exposes the build information
	buildInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metrics.Namespace,
			Subsystem: "build",
			Name:      "info",
			Help:      "stagger build info",
		},
		[]string{"version"})
)

func init() {
	RootCMD.PersistentFlags().IntVar(&LogVerbosity, "log-verbosity", 0, "set logging verbosity")
	RootCMD.PersistentFlags().BoolVar(&ProductionStyleLogging, "log-production", false, "enable production style logging")
	RootCMD.PersistentFlags().StringVar(&MetricsListenAddress, "metrics-listen", ":8080", "prometheus metric exposer listen address")
	RootCMD.PersistentFlags().StringVar(&PProfListenAddress, "pprof-listen", ":6060", "go pprof http listen address")
}

func SetupTelemetryAndLogging() logr.Logger {
	var zlogConfig zap.Config
	if ProductionStyleLogging {
		zlogConfig = zap.NewProductionConfig()
	} else {
		zlogConfig = zap.NewDevelopmentConfig()
	}

	// zlog's log levels are -1*(logr log levels). Ref: https://pkg.go.dev/github.com/go-logr/zapr#hdr-Implementation_Details
	zlogConfig.Level = zap.NewAtomicLevelAt(zapcore.Level(LogVerbosity * -1))
	zlog, _ := zlogConfig.Build()
	logger := zapr.NewLogger(zlog)
	setupPProf(logger, PProfListenAddress)
	setupMetrics(logger, MetricsListenAddress)

	buildInfo.WithLabelValues(version.Build).Set(1)

	return logger
}

func setupPProf(logger logr.Logger, pprofListenAddress string) {
	if pprofListenAddress != "" {
		go func() {
			logger.V(1).Info("starting pprof http handler", "listen", pprofListenAddress)
			err := http.ListenAndServe(pprofListenAddress, nil)
			logger.V(1).Info("pprof http handler terminated", "error", err)
		}()
	}
}

func setupMetrics(logger logr.Logger, metricsListenAddress string) {
	if metricsListenAddress != "" {
		metrics.StartMetricsExposer(metricsListenAddress, logger)
	}
}
