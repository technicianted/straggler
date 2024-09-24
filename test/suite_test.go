// Copyright (c) stagger team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package test

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"stagger/pkg/cmd"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	testEnv        *envtest.Environment
	kubeConfigPath string
	mgr            manager.Manager
	command        *cmd.CMD
	logger         logr.Logger
)

const (
	Namespace = "default"
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Pacer Suite")
}

// GetProjectDir will return the directory where the project is
func GetProjectDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return wd, err
	}
	wd = strings.Replace(wd, "/test/e2e", "", -1)
	return wd, nil
}

// Run executes the provided command within this context
func Run(cmd *exec.Cmd) (string, error) {
	dir, _ := GetProjectDir()
	cmd.Dir = dir
	if err := os.Chdir(cmd.Dir); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "chdir dir: %s\n", err)
	}
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	command := strings.Join(cmd.Args, " ")
	_, _ = fmt.Fprintf(GinkgoWriter, "running: %s\n", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("%s failed with error: (%v) %s", command, err, string(output))
	}
	return string(output), nil
}

// InstallVolcano installs the cert manager bundle.
func InstallVolcano() error {
	url := "https://raw.githubusercontent.com/volcano-sh/volcano/master/installer/volcano-development.yaml"
	cmd := exec.Command("kubectl", "apply", "-f", url)
	if _, err := Run(cmd); err != nil {
		return err
	}
	// Wait for volcano to be ready, which can take time if volcano
	//was re-installed after uninstalling on a cluster.
	cmd = exec.Command("kubectl", "rollout", "status",
		"deployment.apps/volcano-admission",
		"--namespace", "volcano-system",
		"--timeout", "5m",
	)
	_, err := Run(cmd)
	if err != nil {
		return err
	}
	cmd = exec.Command("kubectl", "rollout", "status",
		"deployment.apps/volcano-controllers",
		"--namespace", "volcano-system",
		"--timeout", "5m",
	)
	_, err = Run(cmd)
	if err != nil {
		return err
	}
	cmd = exec.Command("kubectl", "rollout", "status",
		"deployment.apps/volcano-scheduler",
		"--namespace", "volcano-system",
		"--timeout", "5m",
	)
	_, err = Run(cmd)
	return err
}

var _ = BeforeSuite(func() {
	By("bootstrapping test environment")

	Expect(InstallVolcano()).NotTo(HaveOccurred())
	logf.Log.Info("deployed volcano")

	testEnv = &envtest.Environment{
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths:                        []string{filepath.Join("data", "manifest.yaml")},
			LocalServingCertDir:          filepath.Join("data", "tls"),
			LocalServingHost:             "0.0.0.0",
			LocalServingHostExternalName: webhookLocalServingHostExternalName,
		},
	}

	// cfg is defined in this file globally.
	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

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
	By("tearing down the test environment")
	err = testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func setupController(mgr manager.Manager, logger logr.Logger) (*cmd.CMD, error) {
	opts := cmd.NewOptions()
	opts.StaggeringConfigPath = path.Join("data", "stagger-config.yaml")
	return cmd.NewCMDWithManager(mgr, opts, logger)
}
