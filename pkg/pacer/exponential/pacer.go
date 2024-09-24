// Copyright (c) stagger team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package exponential

import (
	"fmt"
	"math"
	"sort"
	"stagger/pkg/pacer/types"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
)

var (
	_ types.Pacer = &pacer{}
)

type pacer struct {
	key    string
	config Config
}

func New(name string, key string, config Config) *pacer {
	return &pacer{
		key:    key,
		config: config,
	}
}

// Pace determines which pods are allowed to be admitted based on exponential pacing.
func (p *pacer) Pace(podClassifications types.PodClassification, logger logr.Logger) ([]corev1.Pod, error) {
	// Enforce MaxStagger limit
	if len(podClassifications.Ready) >= p.config.MaxStagger {
		logger.Info("MaxStagger limit reached, admitting all pending pods")
		return podClassifications.Blocked, nil
	}

	readyCount := len(podClassifications.Ready)
	startingCount := len(podClassifications.Starting)
	blockedCount := len(podClassifications.Blocked)

	allowedCount := calculateAllowedCount(readyCount, startingCount, blockedCount, p.config.MinInitial, p.config.Multiplier)

	// Sort the not admitted pods by creation timestamp in ascending order (earlier pods first)
	sort.Slice(podClassifications.Blocked, func(i, j int) bool {
		return podClassifications.Blocked[i].CreationTimestamp.Time.Before(podClassifications.Blocked[j].CreationTimestamp.Time)
	})

	// Slice the sorted pending pods to allow the determined number of pods
	allowPods := podClassifications.Blocked[:allowedCount]
	totalAdmittedAfterPacing := readyCount + startingCount + len(allowPods)

	logger.Info("pacing decision",
		"ready", readyCount,
		"starting", startingCount,
		"blocked", blockedCount,
		"admitted", len(allowPods),
		"totalAdmittedAfterPacing", totalAdmittedAfterPacing,
	)
	return allowPods, nil
}

func (p *pacer) ID() string {
	return fmt.Sprintf("%T[%s]", p, p.key)
}

func calculateAllowedCount(readyCount, startingCount, blockedCount, minInitial int, multiplier float64) int {
	// Find the next exponential bucket after the ready count
	exponent := 0
	nextTarget := int(float64(minInitial) * math.Pow(multiplier, float64(exponent)))

	for nextTarget <= readyCount {
		exponent++
		nextTarget = int(float64(minInitial) * math.Pow(multiplier, float64(exponent)))
	}

	// Compute the remaining capacity in the bucket after considering starting pods
	totalAdmitted := readyCount + startingCount
	allowedCount := nextTarget - totalAdmitted

	if allowedCount < 0 {
		allowedCount = 0
	}

	// Cap allowedCount to blockedCount
	if allowedCount > blockedCount {
		allowedCount = blockedCount
	}

	return allowedCount
}
