package exponential

import (
	"fmt"
	"testing"
	"time"

	"stagger/pkg/pacer/types"

	"github.com/go-logr/logr"
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

func TestPace(t *testing.T) {
	tests := []struct {
		name                 string
		admittedCount        int
		notAdmittedPodsCount int
		minInitial           int
		multiplier           float64
		maxStagger           int
		expectedAllowedCount int
	}{
		{
			name:                 "Initial Admission",
			admittedCount:        0,
			notAdmittedPodsCount: 10,
			minInitial:           1,
			multiplier:           2,
			maxStagger:           100,
			expectedAllowedCount: 1,
		},
		{
			name:                 "First Exponential Step",
			admittedCount:        1,
			notAdmittedPodsCount: 10,
			minInitial:           1,
			multiplier:           2,
			maxStagger:           100,
			expectedAllowedCount: 1,
		},
		{
			name:                 "Second Exponential Step",
			admittedCount:        2,
			notAdmittedPodsCount: 10,
			minInitial:           1,
			multiplier:           2,
			maxStagger:           100,
			expectedAllowedCount: 2,
		},
		{
			name:                 "Third Exponential Step",
			admittedCount:        4,
			notAdmittedPodsCount: 10,
			minInitial:           1,
			multiplier:           2,
			maxStagger:           100,
			expectedAllowedCount: 4,
		},
		{
			name:                 "Mid Exponential Step",
			admittedCount:        3,
			notAdmittedPodsCount: 10,
			minInitial:           1,
			multiplier:           2,
			maxStagger:           100,
			expectedAllowedCount: 1,
		},
		{
			name:                 "Max Allowed Before Capping",
			admittedCount:        7,
			notAdmittedPodsCount: 10,
			minInitial:           1,
			multiplier:           2,
			maxStagger:           100,
			expectedAllowedCount: 1,
		},
		{
			name:                 "Allowed Count Exceeds Pending Pods",
			admittedCount:        8,
			notAdmittedPodsCount: 5,
			minInitial:           1,
			multiplier:           2,
			maxStagger:           100,
			expectedAllowedCount: 5,
		},
		{
			name:                 "Allowed Count Equals Pending Pods",
			admittedCount:        8,
			notAdmittedPodsCount: 4,
			minInitial:           1,
			multiplier:           2,
			maxStagger:           100,
			expectedAllowedCount: 4,
		},
		{
			name:                 "Large Admitted Count",
			admittedCount:        100,
			notAdmittedPodsCount: 150,
			minInitial:           1,
			multiplier:           2,
			maxStagger:           200,
			expectedAllowedCount: 28,
		},
		{
			name:                 "Non-Power Admitted Count",
			admittedCount:        5,
			notAdmittedPodsCount: 10,
			minInitial:           1,
			multiplier:           2,
			maxStagger:           100,
			expectedAllowedCount: 3,
		},
		{
			name:                 "High Multiplier",
			admittedCount:        10,
			notAdmittedPodsCount: 100,
			minInitial:           2,
			multiplier:           3,
			maxStagger:           100,
			expectedAllowedCount: 8,
		},
		{
			name:                 "MinInitial Greater Than AdmittedCount",
			admittedCount:        1,
			notAdmittedPodsCount: 10,
			minInitial:           2,
			multiplier:           2,
			maxStagger:           100,
			expectedAllowedCount: 2,
		},
		{
			name:                 "Zero Allowed Count Due to No Pending Pods",
			admittedCount:        5,
			notAdmittedPodsCount: 0,
			minInitial:           1,
			multiplier:           2,
			maxStagger:           100,
			expectedAllowedCount: 0,
		},
		{
			name:                 "Fractional Multiplier",
			admittedCount:        3,
			notAdmittedPodsCount: 10,
			minInitial:           1,
			multiplier:           1.5,
			maxStagger:           100,
			expectedAllowedCount: 2,
		},
		{
			name:                 "MinInitial Equal to Admitted Count",
			admittedCount:        2,
			notAdmittedPodsCount: 10,
			minInitial:           2,
			multiplier:           2,
			maxStagger:           100,
			expectedAllowedCount: 2,
		},
		{
			name:                 "Admitted Count Just Below Next Target",
			admittedCount:        7,
			notAdmittedPodsCount: 10,
			minInitial:           1,
			multiplier:           2,
			maxStagger:           100,
			expectedAllowedCount: 1,
		},
		{
			name:                 "Max Stagger Reached (All Pending Pods)",
			admittedCount:        100,
			notAdmittedPodsCount: 50,
			minInitial:           1,
			multiplier:           2,
			maxStagger:           100,
			expectedAllowedCount: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize Config
			config := Config{
				MinInitial: tt.minInitial,
				Multiplier: tt.multiplier,
				MaxStagger: tt.maxStagger,
			}

			// Create a new pacer instance
			p := New("test-pacer", "test-key", config)

			// Generate admitted and not admitted pods
			admittedAndReadyPods := generatePods(tt.admittedCount, "admitted-ready")
			admittedNotReadyPods := generatePods(0, "admitted-not-ready") // Assuming no not ready pods
			notAdmittedPods := generatePods(tt.notAdmittedPodsCount, "not-admitted")

			// Create PodClassification
			podClassifications := types.PodClassification{
				AdmittedAndReadyPods: admittedAndReadyPods,
				AdmittedNotReadyPods: admittedNotReadyPods,
				NotAdmittedPods:      notAdmittedPods,
			}

			// Invoke Pace method
			allowedPods, err := p.Pace(podClassifications, logr.Discard())

			// Check for unexpected errors
			if err != nil {
				t.Fatalf("Pace returned unexpected error: %v", err)
			}

			expected := tt.expectedAllowedCount

			// Determine expected pods based on MaxStagger
			if len(admittedAndReadyPods) >= tt.maxStagger {
				expected = tt.notAdmittedPodsCount
			}

			if len(allowedPods) != expected {
				t.Errorf("Allowed pods count = %d; want %d", len(allowedPods), expected)
			}

			// Verify that the correct pods are admitted
			for i, pod := range allowedPods {
				expectedPod := notAdmittedPods[i]
				if pod.ObjectMeta.Name != expectedPod.ObjectMeta.Name {
					t.Errorf("Allowed pod at index %d = %s; want %s", i, pod.ObjectMeta.Name, expectedPod.ObjectMeta.Name)
				}

			}
		})
	}
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
		AdmittedAndReadyPods: []v1.Pod{},      // No admitted pods initially
		AdmittedNotReadyPods: []v1.Pod{},      // No admitted, not ready pods
		NotAdmittedPods:      notAdmittedPods, // 3 pending pods with different timestamps
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
