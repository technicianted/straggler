// Copyright (c) stagger team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package cmd

import (
	"fmt"
	"net/http"
	"stagger/pkg/blocker"
	"stagger/pkg/config/types"
	"stagger/pkg/controller"
	controllertypes "stagger/pkg/controller/types"
	"stagger/pkg/pacer/exponential"
	pacertypes "stagger/pkg/pacer/types"

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func NewControllerManager(options Options, logger logr.Logger) (manager.Manager, error) {
	webhookOptions := webhook.Options{
		CertDir:  options.TLSDir,
		KeyName:  options.TLSKeyFilename,
		CertName: options.TLSCertFilename,
		Port:     options.TLSListenPort,
	}
	managerOptions := manager.Options{
		LeaderElection:         options.LeaderElection,
		LeaderElectionID:       options.LeaderElectionID,
		Metrics:                server.Options{BindAddress: "0"},
		Logger:                 logger,
		WebhookServer:          webhook.NewServer(webhookOptions),
		HealthProbeBindAddress: options.HealthProbeBindAddress,
	}
	mgr, err := manager.New(
		options.Config,
		managerOptions,
	)
	if err != nil {
		return nil, err
	}
	log.SetLogger(logger)
	mgr.AddReadyzCheck("healthz", func(req *http.Request) error {
		select {
		case _, ok := <-mgr.Elected():
			if !ok {
				return nil
			}
		default:
		}
		return fmt.Errorf("not a leader")
	})
	mgr.AddHealthzCheck("healthz", mgr.GetWebhookServer().StartedChecker())

	return mgr, nil
}

func NewPacerFactory(policy StaggeringPolicy, logger logr.Logger) (pacertypes.PacerFactory, error) {
	switch {
	case policy.Pacer.Exponential != nil:
		config := exponential.Config{
			MinInitial: *policy.Pacer.Exponential.MinInitial,
			MaxStagger: *policy.Pacer.Exponential.MaxStagger,
			Multiplier: *policy.Pacer.Exponential.Multiplier,
		}
		logger.Info("creating exponential pacer", "policy", policy.Name, "config", config)
		return exponential.NewFactory(
			policy.Name,
			config), nil
	default:
		return nil, fmt.Errorf("no pacer configuration specified")
	}
}

func NewGroupClassifier(policies []StaggeringPolicy, logger logr.Logger) (controllertypes.PodClassifier, error) {
	classifier := controller.NewPodClassifier()

	for _, policy := range policies {
		logger.V(1).Info("creating new classifer", "policy", policy.Name, "expression", policy.GroupingExpression)
		pacerFactory, err := NewPacerFactory(policy, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create pacer for %s: %v", policy.Name, err)
		}
		err = classifier.AddConfig(types.StaggerGroup{
			Name:                policy.Name,
			LabelSelector:       policy.LabelSelector,
			BypassLabelSelector: policy.BypassLabelSelector,
			GroupingExpression:  policy.GroupingExpression,
			PacerFactory:        pacerFactory,
		}, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create pod group classifer for %s: %v", policy.Name, err)
		}
	}

	return classifier, nil
}

func NewPodgroupClassifier(mgr manager.Manager, logger logr.Logger) (controllertypes.PodGroupStandingClassifier, error) {
	return controller.NewPodGroupStandingClassifier(mgr.GetClient(), blocker.NewNodeSelectorPodBlocker()), nil
}

func NewRecorderFactory(logger logr.Logger) (controllertypes.ObjectRecorderFactory, error) {
	return nil, nil
}

func RegisterAdmissionController(
	options Options,
	matchPredicate predicate.Predicate,
	mgr manager.Manager,
	classifier controllertypes.PodClassifier,
	podGroupClassifier controllertypes.PodGroupStandingClassifier,
	recorderFactory controllertypes.ObjectRecorderFactory,
	logger logr.Logger,
) error {
	logger.Info("creating admission controller")
	flightTracker := controller.NewFlightTracker(
		mgr.GetClient(),
		options.MaxFlightDuration,
		controller.DefaultStaggerGroupIDLabel,
		logger,
	)
	err := builder.ControllerManagedBy(mgr).
		Named("flight-tracker").
		For(&corev1.Pod{}, builder.WithPredicates(matchPredicate)).
		Complete(flightTracker)
	if err != nil {
		return fmt.Errorf("failed to watch for pods: %v", err)
	}

	admission := controller.NewAdmission(
		classifier,
		podGroupClassifier,
		recorderFactory,
		blocker.NewNodeSelectorPodBlocker(),
		flightTracker,
		options.BypassFailure,
		options.EnableLabel,
	)

	logger.Info("registering admission controller for pods")
	err = builder.WebhookManagedBy(mgr).
		For(&corev1.Pod{}).
		WithDefaulter(admission).
		Complete()
	if err != nil {
		return fmt.Errorf("failed to register pod admission: %v", err)
	}

	logger.Info("registering admission controller for jobs")
	err = builder.WebhookManagedBy(mgr).
		For(&batchv1.Job{}).
		WithDefaulter(admission).
		Complete()
	if err != nil {
		return fmt.Errorf("failed to register pod admission: %v", err)
	}

	return nil
}

func RegisterReconciler(
	options Options,
	matchPredicate predicate.Predicate,
	mgr manager.Manager,
	classifier controllertypes.PodClassifier,
	podGroupClassifier controllertypes.PodGroupStandingClassifier,
	logger logr.Logger,
) error {
	reconciler := controller.NewReconciler(
		mgr.GetClient(),
		classifier,
		podGroupClassifier)
	err := builder.ControllerManagedBy(mgr).
		Named("reconciler").
		For(&corev1.Pod{}, builder.WithPredicates(matchPredicate)).
		Complete(reconciler)
	if err != nil {
		return fmt.Errorf("failed to watch for pods: %v", err)
	}

	return nil
}

func GetMatchLabelsPredicate(options Options, config Config, logger logr.Logger) (predicate.Predicate, error) {
	expressions := []metav1.LabelSelectorRequirement{
		{
			Key:      options.EnableLabel,
			Operator: metav1.LabelSelectorOpExists,
		},
	}
	for _, policy := range config.StaggeringPolicies {
		for label := range policy.LabelSelector {
			expressions = append(expressions, metav1.LabelSelectorRequirement{
				Key:      label,
				Operator: metav1.LabelSelectorOpExists,
			})
		}
	}

	logger.Info("extracted reconciliation match expressions", "expression", expressions)

	return predicate.LabelSelectorPredicate(
		metav1.LabelSelector{
			MatchExpressions: expressions,
		})
}

func CreateKubernetesConfig(opts KubernetesOptions) (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags(opts.MasterURL, opts.KubeConfigPath)
		if err != nil {
			return nil, fmt.Errorf("error building kubeconfig: %v", err)
		}
	}

	return config, err
}
