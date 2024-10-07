// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type CMD struct {
	options Options

	mgr                   manager.Manager
	shutdownContext       context.Context
	shutdownContextCancel context.CancelFunc
	shutdownCompleteChan  chan struct{}
}

func NewCMDWithManager(mgr manager.Manager, options Options, logger logr.Logger) (*CMD, error) {
	config, err := LoadConfig(options.StaggeringConfigPath, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to load configs: %v", err)
	}

	classifier, err := NewGroupClassifier(config.StaggeringPolicies, logger)
	if err != nil {
		return nil, err
	}

	blocker, err := NewBlocker(options)
	if err != nil {
		return nil, err
	}
	podGroupClassifier, err := NewPodgroupClassifier(mgr, blocker, logger)
	if err != nil {
		return nil, err
	}

	recorderFactory, err := NewRecorderFactory(logger)
	if err != nil {
		return nil, err
	}

	matchPredicate, err := GetMatchLabelsPredicate(options, config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to get match predicates for reconciler: %v", err)
	}
	if err := RegisterReconciler(
		options,
		matchPredicate,
		mgr,
		classifier,
		podGroupClassifier,
		logger,
	); err != nil {
		return nil, err
	}
	if err := RegisterAdmissionController(
		options,
		matchPredicate,
		mgr,
		blocker,
		classifier,
		podGroupClassifier,
		recorderFactory,
		logger,
	); err != nil {
		return nil, err
	}

	return &CMD{
		options: options,
		mgr:     mgr,
	}, nil
}

func NewCMD(options Options, logger logr.Logger) (*CMD, error) {
	kubernetesConfig, err := CreateKubernetesConfig(options.KubernetesOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes config: %v", err)
	}
	options.Config = kubernetesConfig

	mgr, err := NewControllerManager(options, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create controller manager: %v", err)
	}

	return NewCMDWithManager(mgr, options, logger)
}

func (c *CMD) Start(logger logr.Logger) error {
	logger.Info("starting")
	c.shutdownCompleteChan = make(chan struct{})
	c.shutdownContext, c.shutdownContextCancel = context.WithCancel(context.Background())
	go func() {
		err := c.mgr.Start(c.shutdownContext)
		if err != nil {
			logger.Info("failed to start controller manager", "error", err)
		}
		close(c.shutdownCompleteChan)
	}()

	logger.Info("waiting for caches to be ready")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if !c.mgr.GetCache().WaitForCacheSync(ctx) {
		return fmt.Errorf("failed to sync caches: %v", ctx.Err())
	}

	return nil
}

func (c *CMD) Stop(logger logr.Logger) error {
	logger.Info("shutting down")

	if c.shutdownCompleteChan != nil {
		c.shutdownContextCancel()
		logger.Info("waiting for shutdown")
		<-c.shutdownCompleteChan
		logger.Info("shutdown completed")
	}

	return nil
}
