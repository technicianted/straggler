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
	name   string
	key    string
	config Config
}

func New(name string, key string, config Config) *pacer {
	return &pacer{
		name:   name,
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

	admittedCount := len(podClassifications.Ready) + len(podClassifications.Starting)

	allowedCount := calculateAllowedCount(admittedCount, len(podClassifications.Blocked), p.config.MinInitial, p.config.Multiplier)

	// Sort the not admitted pods by creation timestamp in ascending order (earlier pods first)
	sort.Slice(podClassifications.Blocked, func(i, j int) bool {
		return podClassifications.Blocked[i].CreationTimestamp.Time.Before(podClassifications.Blocked[j].CreationTimestamp.Time)
	})

	// Slice the sorted pending pods to allow the determined number of pods
	allowPods := podClassifications.Blocked[:allowedCount]
	totalAdmittedAfterPacing := admittedCount + len(allowPods)

	logger.Info("pacing decision",
		"ready", len(podClassifications.Ready),
		"starting", len(podClassifications.Starting),
		"blocked", len(podClassifications.Blocked),
		"admitted", len(allowPods),
		"totalAdmittedAfterPacing", totalAdmittedAfterPacing,
	)
	return allowPods, nil
}

func (p *pacer) String() string {
	return fmt.Sprintf("%T[%s]: %s", p, p.name, p.key)
}

// calculateAllowedCount computes the number of pods to admit based on exponential pacing.
// Parameters:
// - admittedCount: Total number of currently admitted pods.
// - notAdmittedPodsCount: Number of pods pending admission.
// - minInitial: Minimum number of pods to admit initially.
// - multiplier: Factor by which to exponentially increase the batch size.
//
// Returns:
// - allowedCount: Number of pods allowed to be admitted in the current pacing cycle.
func calculateAllowedCount(admittedCount, notAdmittedPodsCount, minInitial int, multiplier float64) int {
	// Start with the minimum initial number of pods allowed
	allowedCount := minInitial

	if admittedCount > 0 {
		// Calculate the exponent such that minInitial * multiplier^exponent > admittedCount
		exponent := math.Floor(math.Log(float64(admittedCount)/float64(minInitial)) / math.Log(multiplier))

		// Ensure exponent is non-negative
		if exponent < 0 {
			exponent = 0
		}

		// Calculate the next target based on the exponent
		nextTarget := int(float64(minInitial) * math.Pow(multiplier, exponent))

		// If nextTarget <= admittedCount, increment exponent until nextTarget > admittedCount
		for nextTarget <= admittedCount {
			exponent++
			nextTarget = int(float64(minInitial) * math.Pow(multiplier, exponent))
		}

		// Determine the allowed count as the difference between nextTarget and admittedCount
		allowedCount = nextTarget - admittedCount

		// Ensure allowedCount is at least minInitial
		if allowedCount < minInitial {
			allowedCount = minInitial
		}
	}

	// Cap allowedCount to notAdmittedPodsCount
	if allowedCount > notAdmittedPodsCount {
		allowedCount = notAdmittedPodsCount
	}

	return allowedCount
}
