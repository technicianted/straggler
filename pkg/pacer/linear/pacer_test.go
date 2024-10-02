// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package linear

import (
	"straggler/pkg/pacer/types"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestLinearPacerAllow(t *testing.T) {
	pacer := New(
		"key",
		Config{
			MaxStagger: 12,
			Step:       5,
		})

	// initial case, should allow step
	allowed, err := pacer.Pace(types.PodClassification{
		Ready:    []corev1.Pod{},
		Starting: []corev1.Pod{},
		Blocked:  []corev1.Pod{{}},
	},
		logr.Discard(),
	)
	require.NoError(t, err)
	require.Len(t, allowed, 1)
	// > step
	allowed, err = pacer.Pace(types.PodClassification{
		Ready:    []corev1.Pod{},
		Starting: []corev1.Pod{},
		Blocked:  []corev1.Pod{{}, {}, {}, {}, {}, {}},
	},
		logr.Discard(),
	)
	require.NoError(t, err)
	require.Len(t, allowed, 5)

	// ready < step
	allowed, err = pacer.Pace(types.PodClassification{
		Ready:    []corev1.Pod{{}, {}, {}, {}},
		Starting: []corev1.Pod{},
		Blocked:  []corev1.Pod{{}},
	},
		logr.Discard(),
	)
	require.NoError(t, err)
	require.Len(t, allowed, 1)
}

func TestLinearPacerBlock(t *testing.T) {
	pacer := New(
		"key",
		Config{
			MaxStagger: 12,
			Step:       5,
		})

	// ready < step but ready + starting on boundary
	allowed, err := pacer.Pace(types.PodClassification{
		Ready:    []corev1.Pod{{}, {}, {}},
		Starting: []corev1.Pod{{}, {}},
		Blocked:  []corev1.Pod{},
	},
		logr.Discard(),
	)
	require.NoError(t, err)
	require.Len(t, allowed, 0)
	// ready < step but ready + starting > boundary
	allowed, err = pacer.Pace(types.PodClassification{
		Ready:    []corev1.Pod{{}, {}, {}},
		Starting: []corev1.Pod{{}, {}, {}},
		Blocked:  []corev1.Pod{},
	},
		logr.Discard(),
	)
	require.NoError(t, err)
	require.Len(t, allowed, 0)
}
