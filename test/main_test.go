package test

import (
	"context"
	"net"
	"os"
	"runtime"
	"testing"

	"github.com/foxcpp/go-mockdns"
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
	webhookLocalServingHostExternalName string
)

// setupTestDnsServer sets up a mock DNS server that resolves some test entries tio
func setupTestDnsServer(zones map[string]mockdns.Zone) func() {
	srv, _ := mockdns.NewServer(zones, false)
	srv.PatchNet(net.DefaultResolver)

	return func() {
		defer srv.Close()
		mockdns.UnpatchNet(net.DefaultResolver)
	}
}

func TestMain(m *testing.M) {
	// zlog's log levels are -1*(logr log levels). Ref: https://pkg.go.dev/github.com/go-logr/zapr#hdr-Implementation_Details
	logger = zap.New(zap.WriteTo(ginkgo.GinkgoWriter), zap.UseDevMode(true), zap.Level(zapcore.Level(-100)))
	logf.SetLogger(logger)

	// only do this on mac
	// check is os is darwin
	if runtime.GOOS == "darwin" {
		webhookLocalServingHostExternalName = "host.docker.internal"
		// setup dns server
		teardownDnsServer := setupTestDnsServer(map[string]mockdns.Zone{
			// The webhook is called from within the api server running in the kind cluster.
			// And since the webhook is running locally, we need to use the host.docker.internal to the host machine's IP.
			// But since we will generate a CA and a cert for the webhook using TinyCA, we need to make sure that the host.docker.internal is resolvable.
			// So we add a fake entry for host.docker.internal to any IP.
			"host.docker.internal.": {
				A: []string{"1.2.3.4"},
			},
		})
		defer teardownDnsServer()
	} else {
		webhookLocalServingHostExternalName = "172.17.0.1"
	}

	kindEnv, _ := env.NewFromFlags()
	kindClusterName := envconf.RandomName("kind", 16)

	kindEnv.Setup(
		envfuncs.CreateCluster(kind.NewProvider(), kindClusterName),
		func(ctx context.Context, c *envconf.Config) (context.Context, error) {
			kubeConfigPath = c.KubeconfigFile()
			os.Setenv("USE_EXISTING_CLUSTER", "true")
			os.Setenv("KUBECONFIG", kubeConfigPath)
			logger.Info("To import the kubeconfig, run the following command", "command", "export KUBECONFIG="+kubeConfigPath)
			return ctx, nil
		},
	)

	kindEnv.Finish(
		envfuncs.ExportClusterLogs(kindClusterName, "./logs"),
		envfuncs.DestroyCluster(kindClusterName),
	)

	os.Exit(kindEnv.Run(m))
}
