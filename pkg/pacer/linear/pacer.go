// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package linear

import (
	"fmt"
	"math"
	"straggler/pkg/pacer/types"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
)

type Config struct {
	// Maximum number of staggered pods after which it's disabled.
	MaxStagger int
	// Number of pods to add at each step.
	Step int
}

// Linear pacer is an artithmatic pacer that unblocks pods in predefined
// or steps.
// For example: 5 => 10 => 15 => ...
type pacer struct {
	key    string
	config Config
}

func New(key string, config Config) *pacer {
	return &pacer{
		key:    key,
		config: config,
	}
}

func (p *pacer) Pace(podClassifications types.PodClassification, logger logr.Logger) (allowPods []corev1.Pod, err error) {
	// Enforce MaxStagger limit
	if len(podClassifications.Ready) >= p.config.MaxStagger {
		logger.V(1).Info("MaxStagger limit reached, admitting all pending pods")
		return podClassifications.Blocked, nil
	}

	readyCount := len(podClassifications.Ready)
	startingCount := len(podClassifications.Starting)
	blockedCount := len(podClassifications.Blocked)

	remainder := readyCount % p.config.Step
	allowCount := p.config.Step - remainder
	if allowCount <= startingCount {
		return nil, nil
	}

	toAllow := int(math.Min(float64(blockedCount), float64(allowCount)))
	return podClassifications.Blocked[:toAllow], nil
}

func (p *pacer) ID() string {
	return fmt.Sprintf("%T[%s]", p, p.key)
}
