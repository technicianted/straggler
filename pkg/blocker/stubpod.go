// Copyright (c) stagger team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package blocker

import (
	"fmt"
	"stagger/pkg/blocker/types"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
)

var _ types.PodBlocker = &stubPod{}

type stubPod struct {
	containerImage string
}

func NewStubPod(containerImage string) types.PodBlocker {
	return &stubPod{
		containerImage: containerImage,
	}
}

func (b *stubPod) Block(podSpec *corev1.PodSpec, logger logr.Logger) error {
	for i, container := range podSpec.InitContainers {
		container.Image = b.containerImage
		container.Command = nil
		container.Args = []string{"initcontainer"}
		container.VolumeMounts = nil
		podSpec.InitContainers[i] = container
	}
	// these will never start, just stubs to prevent image pulls
	for i, container := range podSpec.Containers {
		container.Image = b.containerImage
		container.Command = nil
		container.Args = nil
		container.VolumeMounts = nil
		podSpec.Containers[i] = container
	}
	podSpec.InitContainers = append(podSpec.InitContainers, corev1.Container{
		Name:  "stagger",
		Image: b.containerImage,
		Args:  []string{"container"},
	})

	return nil
}

func (b *stubPod) Unblock(podSpec *corev1.PodSpec, logger logr.Logger) error {
	return fmt.Errorf("not supported")
}

func (b *stubPod) IsBlocked(podSpec *corev1.PodSpec) bool {
	for _, container := range podSpec.InitContainers {
		if container.Name == "stagger" &&
			container.Image == b.containerImage {
			return true
		}
	}

	return false
}
