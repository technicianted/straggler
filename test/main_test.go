package test

import (
	"context"
	"net"
	"os"
	"testing"

	"github.com/foxcpp/go-mockdns"
	"github.com/go-logr/logr"
	"github.com/onsi/ginkgo/v2"
	"go.uber.org/zap/zapcore"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/support/kind"
)

var (
	logger logr.Logger
)

func TestMain(m *testing.M) {
	// zlog's log levels are -1*(logr log levels). Ref: https://pkg.go.dev/github.com/go-logr/zapr#hdr-Implementation_Details
	logger = zap.New(zap.WriteTo(ginkgo.GinkgoWriter), zap.UseDevMode(true), zap.Level(zapcore.Level(-100)))
	logf.SetLogger(logger)
	logger.V(10).Info("=-=-=-=-=-=-= Starting tests =-=-=-=-=-=-")

	srv, _ := mockdns.NewServer(map[string]mockdns.Zone{
		"host.docker.internal.": {
			A: []string{"1.2.3.4"},
		},
	}, false)
	defer srv.Close()

	srv.PatchNet(net.DefaultResolver)
	defer mockdns.UnpatchNet(net.DefaultResolver)

	// check if ls ./bin/kubebuilder/ directory exiss and has files
	path := "../bin/kubebuilder/"
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		logger.Error(err, "kubebuilder directory does not exist. Please run make kubeassets")
		os.Exit(1)
	}

	if err != nil {
		logger.Error(err, "failed to check kubebuilder directory")
		os.Exit(1)
	}

	files, err := os.ReadDir(path)
	if err != nil {
		logger.Error(err, "failed to read kubebuilder directory")
		os.Exit(1)
	}

	// check if the directory is empty
	if len(files) == 0 {
		logger.Error(err, "kubebuilder directory is empty. Please run make kubeassets")
		os.Exit(1)
	}

	// set KUBEBUILDER_ASSETS env variable to the path
	err = os.Setenv("KUBEBUILDER_ASSETS", path)
	if err != nil {
		logger.Error(err, "failed to set KUBEBUILDER_ASSETS env variable")
		os.Exit(1)
	}

	testenv, _ := env.NewFromFlags()
	kindClusterName := envconf.RandomName("kind", 16)

	testenv.Setup(
		envfuncs.CreateCluster(kind.NewProvider(), kindClusterName),
		func(ctx context.Context, c *envconf.Config) (context.Context, error) {
			kubeConfigPath = c.KubeconfigFile()
			os.Setenv("USE_EXISTING_CLUSTER", "true")
			os.Setenv("KUBECONFIG", kubeConfigPath)
			return ctx, nil
		},
	)

	testenv.Finish(
		envfuncs.ExportClusterLogs(kindClusterName, "./logs"),
		envfuncs.DestroyCluster(kindClusterName),
	)
	os.Exit(testenv.Run(m))
}
