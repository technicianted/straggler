package test

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"stagger/pkg/cmd"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	cfg     *rest.Config
	testEnv *envtest.Environment
	ctx     context.Context
	cancel  context.CancelFunc

	kubeConfigPath string

	namespace = "default"

	mgr     manager.Manager
	command *cmd.CMD
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Webhook Suite")
}

func setupController(mgr manager.Manager, logger logr.Logger) (*cmd.CMD, error) {
	opts := cmd.NewOptions()
	opts.StaggeringConfigPath = path.Join("data", "stagger-config.yaml")
	return cmd.NewCMDWithManager(mgr, opts, logger)
}

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")

	var err error

	testEnv = &envtest.Environment{
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths:                        []string{filepath.Join("data", "manifest.yaml")},
			LocalServingCertDir:          filepath.Join("data", "tls"),
			LocalServingHostExternalName: "host.docker.internal",
		},
	}

	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	// create a temp path for the kubeconfig
	kubeConfigPath = CreateKubeconfigFileForRestConfig(*cfg)
	logf.Log.Info("kubeconfig path", "path", kubeConfigPath)
	// write testenv.config to the kubeconfig path

	// // Register core Kubernetes APIs (usually already registered, but can be explicit)
	// err = corev1.AddToScheme(scheme)
	// Expect(err).NotTo(HaveOccurred())

	// // **Register the Deployment kind**
	// err = appsv1.AddToScheme(scheme)
	// Expect(err).NotTo(HaveOccurred())

	// start webhook server using Manager
	webhookInstallOptions := &testEnv.WebhookInstallOptions
	mgr, err = ctrl.NewManager(cfg, ctrl.Options{
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    webhookInstallOptions.LocalServingHost,
			Port:    webhookInstallOptions.LocalServingPort,
			CertDir: webhookInstallOptions.LocalServingCertDir,
		}),
		LeaderElection: false,
		Metrics:        metricsserver.Options{BindAddress: "0"},
	})
	Expect(err).NotTo(HaveOccurred())

	command, err = setupController(mgr, logf.Log)
	Expect(err).NotTo(HaveOccurred())

	// log all the server options
	logf.Log.Info("Webhook server options", "host", webhookInstallOptions.LocalServingHost, "port", webhookInstallOptions.LocalServingPort, "certDir", webhookInstallOptions.LocalServingCertDir)
	Expect(err).NotTo(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err := command.Start(logf.Log)
		Expect(err).NotTo(HaveOccurred())
	}()

	// wait for the webhook server to get ready
	Eventually(func() error {
		// check if the server is ready
		// time.Sleep(time.Minute * 2)
		return nil
	}).Should(Succeed())

})

var _ = AfterSuite(func() {
	err := command.Stop(logf.Log)
	Expect(err).NotTo(HaveOccurred())
	cancel()
	By("tearing down the test environment")
	err = testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func CreateKubeconfigFileForRestConfig(restConfig rest.Config) string {
	clusters := make(map[string]*clientcmdapi.Cluster)
	clusters["default-cluster"] = &clientcmdapi.Cluster{
		Server:                   restConfig.Host,
		CertificateAuthorityData: restConfig.CAData,
	}
	contexts := make(map[string]*clientcmdapi.Context)
	contexts["default-context"] = &clientcmdapi.Context{
		Cluster:  "default-cluster",
		AuthInfo: "default-user",
	}
	authinfos := make(map[string]*clientcmdapi.AuthInfo)
	authinfos["default-user"] = &clientcmdapi.AuthInfo{
		ClientCertificateData: restConfig.CertData,
		ClientKeyData:         restConfig.KeyData,
	}
	clientConfig := clientcmdapi.Config{
		Kind:           "Config",
		APIVersion:     "v1",
		Clusters:       clusters,
		Contexts:       contexts,
		CurrentContext: "default-context",
		AuthInfos:      authinfos,
	}
	kubeConfigFile, _ := os.CreateTemp("", "kubeconfig")
	_ = clientcmd.WriteToFile(clientConfig, kubeConfigFile.Name())
	return kubeConfigFile.Name()
}
