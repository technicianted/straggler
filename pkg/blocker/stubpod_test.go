// Copyright (c) stagger team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package blocker

import (
	"testing"

	"github.com/go-logr/zapr"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

func TestStubPodSuccess(t *testing.T) {
	zlog, _ := zap.NewDevelopment()
	logger := zapr.NewLogger(zlog)

	pod := corev1.Pod{
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{
				{
					Name:    "init1",
					Image:   "image1",
					Command: []string{"command1"},
					Args:    []string{"args1"},
				},
			},
			Containers: []corev1.Container{
				{
					Name:    "container1",
					Image:   "image1",
					Command: []string{"command1"},
					Args:    []string{"args1"},
				},
			},
		},
	}

	blocker := NewStubPod("staggerimage")
	err := blocker.Block(&pod.Spec, logger)
	require.NoError(t, err)
	require.Len(t, pod.Spec.InitContainers, 2)
	initContainer := pod.Spec.InitContainers[0]
	require.Nil(t, initContainer.Command)
	require.Equal(t, "staggerimage", initContainer.Image)
	initContainer = pod.Spec.InitContainers[1]
	require.Equal(t, "staggerimage", initContainer.Image)

	require.Len(t, pod.Spec.Containers, 1)
	container := pod.Spec.Containers[0]
	require.Nil(t, container.Command)
	require.Equal(t, "staggerimage", container.Image)

	blocked := blocker.IsBlocked(&pod.Spec)
	require.True(t, blocked)

	pod.Spec.InitContainers[1].Image = "someotherimage"
	blocked = blocker.IsBlocked(&pod.Spec)
	require.False(t, blocked)
}
