// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package exponential

import (
	"fmt"
	"math"
	"sort"
	"straggler/pkg/pacer/types"

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

func New(key string, config Config) *pacer {
	return &pacer{
		key:    key,
		config: config,
	}
}

// Pace determines which pods are allowed to be admitted based on exponential pacing.
func (p *pacer) Pace(podClassifications types.PodClassification, logger logr.Logger) ([]corev1.Pod, error) {
	// Enforce MaxStagger limit
	if len(podClassifications.Ready) >= p.config.MaxStagger {
		logger.V(1).Info("MaxStagger limit reached, admitting all pending pods")
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
	// If there are no ready pods, we should admit the minimum initial count
	if readyCount == 0 {
		return min(max(0, minInitial-startingCount), blockedCount)
	}

	// Calculate the desired count based on the ready count
	desiredCount := int(math.Ceil(float64(readyCount) * multiplier))

	totalAdmitted := readyCount + startingCount

	// Now lets remove the admitted pods from the desired count to get the allowed count
	allowedCount := max(0, desiredCount-totalAdmitted)

	// We should not admit more than the number of blocked pods
	allowedCount = min(allowedCount, blockedCount)

	return allowedCount
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
