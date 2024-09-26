// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package types

import (
	"context"

	configtypes "straggler/pkg/config/types"
	pacertypes "straggler/pkg/pacer/types"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

//go:generate mockgen -package mocks -destination ../mocks/blockers.go -source $GOFILE
//go:generate mockgen -package mocks -destination ../mocks/client.go sigs.k8s.io/controller-runtime/pkg/client Client,SubResourceClient

type ObjectRecorder interface {
	Normalf(reason, format string, args ...interface{})
	Warnf(reason, format string, args ...interface{})
	Logf(logger logr.Logger, v int, reason, format string, args ...interface{})
}

type ObjectRecorderFactory interface {
	RecorderForRootControllerOrNull(ctx context.Context, object runtime.Object, logger logr.Logger) ObjectRecorder
	RecorderForRootController(ctx context.Context, object runtime.Object, logger logr.Logger) (ObjectRecorder, error)
}

// Pod classification result.
type PodClassification struct {
	// Unique ID to identify this particular pacer instance. It
	// can be used later to retrieve it.
	ID string
	// Pacer used for staggering this pod.
	Pacer pacertypes.Pacer
}

// Classify a pod to a staggering pacer.
type PodClassifier interface {
	// Classify a pod to a staggering group. If pod does not belong to any group
	// nil is returned.
	Classify(podMeta metav1.ObjectMeta, podSpec corev1.PodSpec, logger logr.Logger) (*PodClassification, error)
	ClassifyByGroupID(groupID string, logger logr.Logger) (*PodClassification, error)
}

// Interface to provide classification of all pods within a staggering group.
type PodGroupStandingClassifier interface {
	ClassifyPodGroup(ctx context.Context, groupID string, logger logr.Logger) (ready, starting, blocked []corev1.Pod, err error)
}

// Configuration interface for a pod classifier.
type PodClassifierConfigurator interface {
	AddConfig(config configtypes.StaggerGroup, logger logr.Logger) error
	RemoveConfig(name string, logger logr.Logger) error
	UpdateConfig(config configtypes.StaggerGroup, logger logr.Logger) error
}

// An implementation that is used by the admission controller to minimize
// race admitted pods and committed pods.
// It is assumed that it is best effort.
type AdmissionFlightTracker interface {
	Track(key string, object metav1.ObjectMeta, logger logr.Logger) error
	WaitOne(ctx context.Context, key string, logger logr.Logger) error
}
