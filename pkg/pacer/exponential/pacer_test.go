package exponential

import (
	"fmt"
	"testing"
	"time"

	"stagger/pkg/pacer/types"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Helper function to generate dummy pods with unique names.
func generatePods(count int, prefix string) []v1.Pod {
	pods := make([]v1.Pod, count)
	for i := 0; i < count; i++ {
		pods[i].ObjectMeta.Name = fmt.Sprintf("%s-%d", prefix, i+1)
	}
	return pods
}

func TestPace_SortsByCreationTimestamp(t *testing.T) {
	config := Config{
		MinInitial: 1,
		Multiplier: 2,
		MaxStagger: 100,
	}

	p := New("test-pacer", "test-key", config)

	// Create pods with different creation timestamps
	now := metav1.Now()
	earlier := metav1.NewTime(now.Add(-10 * time.Minute))
	later := metav1.NewTime(now.Add(10 * time.Minute))

	notAdmittedPods := []v1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-later", CreationTimestamp: later},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-earlier", CreationTimestamp: earlier},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-now", CreationTimestamp: now},
		},
	}

	podClassifications := types.PodClassification{
		Ready:    []v1.Pod{},      // No admitted pods initially
		Starting: []v1.Pod{},      // No admitted, not ready pods
		Blocked:  notAdmittedPods, // 3 pending pods with different timestamps
	}

	// Invoke Pace method
	allowedPods, err := p.Pace(podClassifications, logr.Discard())
	if err != nil {
		t.Fatalf("Pace returned unexpected error: %v", err)
	}

	// We expect only the first pod (with the earliest creation timestamp) to be admitted
	expectedPodName := "pod-earlier"

	if len(allowedPods) != 1 {
		t.Errorf("Expected 1 pod to be admitted, but got %d", len(allowedPods))
	}

	if allowedPods[0].ObjectMeta.Name != expectedPodName {
		t.Errorf("Expected admitted pod to be %s, but got %s", expectedPodName, allowedPods[0].ObjectMeta.Name)
	}
}

func TestPace(t *testing.T) {
	tests := []struct {
		name            string
		readyCount      int
		startingCount   int
		blockedCount    int
		minInitial      int
		multiplier      float64
		maxStagger      int
		expectedAllowed int
	}{
		{
			name:            "All counts zero",
			readyCount:      0,
			startingCount:   0,
			blockedCount:    0,
			minInitial:      1,
			multiplier:      2.0,
			maxStagger:      16,
			expectedAllowed: 0,
		},
		{
			name:            "No ready pods, some blocked pods",
			readyCount:      0,
			startingCount:   0,
			blockedCount:    5,
			minInitial:      1,
			multiplier:      2.0,
			maxStagger:      16,
			expectedAllowed: 1,
		},
		{
			name:            "Some ready pods, no starting pods, some blocked pods",
			readyCount:      3,
			startingCount:   0,
			blockedCount:    10,
			minInitial:      1,
			multiplier:      2.0,
			maxStagger:      16,
			expectedAllowed: 1,
		},
		{
			name:            "Ready and starting pods, minInitial 2, multiplier 1.5",
			readyCount:      4,
			startingCount:   2,
			blockedCount:    5,
			minInitial:      2,
			multiplier:      1.5,
			maxStagger:      16,
			expectedAllowed: 0,
		},
		{
			name:            "Allowed count becomes negative",
			readyCount:      5,
			startingCount:   2,
			blockedCount:    3,
			minInitial:      2,
			multiplier:      2.0,
			maxStagger:      16,
			expectedAllowed: 1,
		},
		{
			name:            "Allowed count exceeds blocked pods",
			readyCount:      2,
			startingCount:   1,
			blockedCount:    1,
			minInitial:      1,
			multiplier:      2.0,
			maxStagger:      16,
			expectedAllowed: 1,
		},
		{
			name:            "Large blockedCount with minInitial 5",
			readyCount:      0,
			startingCount:   0,
			blockedCount:    20,
			minInitial:      5,
			multiplier:      2.0,
			maxStagger:      16,
			expectedAllowed: 5,
		},
		{
			name:            "Large numbers for ready, starting, blocked counts",
			readyCount:      50,
			startingCount:   20,
			blockedCount:    100,
			minInitial:      10,
			multiplier:      2.0,
			maxStagger:      256,
			expectedAllowed: 10,
		},
		{
			name:            "MaxStagger reached",
			readyCount:      16,
			startingCount:   3,
			blockedCount:    7,
			minInitial:      1,
			multiplier:      2.0,
			maxStagger:      16,
			expectedAllowed: 7,
		},
		{
			name:            "Ready and starting exceeds exponential bucket",
			readyCount:      3,
			startingCount:   3,
			blockedCount:    1,
			minInitial:      1,
			multiplier:      2.0,
			maxStagger:      16,
			expectedAllowed: 0,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			// Initialize Config
			config := Config{
				MinInitial: tt.minInitial,
				Multiplier: tt.multiplier,
				MaxStagger: tt.maxStagger,
			}
			readyPods := generatePods(tt.readyCount, "ready")
			startingPods := generatePods(tt.startingCount, "starting")
			blockedPods := generatePods(tt.blockedCount, "blocked")

			pacer := New("test-pacer", "test-key", config)
			allowed, err := pacer.Pace(types.PodClassification{
				Ready:    readyPods,
				Starting: startingPods,
				Blocked:  blockedPods,
			}, logr.Discard())
			require.NoError(t, err)
			require.Len(t, allowed, tt.expectedAllowed, "unexpected allowed count, expected %d, got %d", tt.expectedAllowed, len(allowed))
		})
	}
}
