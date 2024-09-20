package cmd

import (
	"os"
	"stagger/pkg/controller"
	"time"

	"k8s.io/client-go/rest"
)

type LeaderElectionOptions struct {
	LeaderElection   bool   `cliArgName:"kubernetes-leader-election" cliArgDescription:"enable leader election" cliArgGroup:"Kubernetes"`
	LeaderElectionID string `cliArgName:"kubernetes-leader-election-id" cliArgDescription:"id to use for kubernetes leader election" cliArgGroup:"Kubernetes"`
}

type KubernetesOptions struct {
	LeaderElectionOptions
	KubeConfigPath string `cliArgName:"kubernetes-kubeconfig" cliArgDescription:"path to kubeconfig file" cliArgGroup:"Kubernetes"`
	MasterURL      string `cliArgName:"kubernetes-master-url" cliArgDescription:"api server url" cliArgGroup:"Kubernetes"`

	RegisterHeatlhChecks   bool
	HealthProbeBindAddress string

	Config *rest.Config
}

type Options struct {
	KubernetesOptions

	StaggeringConfigPath   string        `cliArgName:"staggering-config-path" cliArgDescription:"path to staggering config yaml file" cliArgGroup:"Staggering"`
	BypassFailure          bool          `cliArgName:"staggering-bypass-errors" cliArgDescription:"do not block admission on errors" cliArgGroup:"Staggering"`
	EnableLabel            string        `cliArgName:"staggering-enable-label" cliArgDescription:"pod label to enable staggering behavior" cliArgGroup:"Staggering"`
	MaxFlightDuration      time.Duration `cliArgName:"staggering-max-pod-flight-duration" cliArgDescription:"maximum time to wait for a pod from admission to reconciliation after which it is assumed committed" cliArgGroup:"Staggering"`
	TLSDir                 string        `cliArgName:"tls-dir" cliArgDescription:"dir to look for tls pem files" cliArgGroup:"TLS"`
	TLSKeyFilename         string        `cliArgName:"tls-key-filename" cliArgDescription:"path to tls key pem" cliArgGroup:"TLS"`
	TLSCertFilename        string        `cliArgName:"tls-cert-filename" cliArgDescription:"path to tls certificate pem" cliArgGroup:"TLS"`
	TLSListenPort          int           `cliArgName:"tls-port" cliArgDescription:"port to listen on for webhook admission requests" cliArgGroup:"TLS"`
	HealthProbeBindAddress string        `cliArgName:"health-probe-bind-address" cliArgDescription:"address to bind on for http health server" cliArgGroup:"Health"`
}

func NewKubernetesOptions() KubernetesOptions {
	return KubernetesOptions{
		LeaderElectionOptions: NewLeaderElectionOptions(),
		KubeConfigPath:        os.Getenv("KUBECONFIG"),
	}
}

func NewLeaderElectionOptions() LeaderElectionOptions {
	return LeaderElectionOptions{
		LeaderElection:   true,
		LeaderElectionID: "stagger",
	}
}

func NewOptions() Options {
	return Options{
		KubernetesOptions:      NewKubernetesOptions(),
		BypassFailure:          true,
		EnableLabel:            controller.DefaultEnableLabel,
		MaxFlightDuration:      1000 * time.Millisecond,
		TLSDir:                 ".",
		TLSKeyFilename:         "tls.key",
		TLSCertFilename:        "tls.crt",
		TLSListenPort:          9443,
		HealthProbeBindAddress: ":9444",
	}
}
