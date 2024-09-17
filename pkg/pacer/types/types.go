package types

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
)

//go:generate mockgen -package mocks -destination ../mocks/pacer.go -source $GOFILE

// PodClassification categorizes pods based on their admission and readiness status.
type PodClassification struct {
	Ready    []corev1.Pod
	Starting []corev1.Pod
	Blocked  []corev1.Pod
}

type Pacer interface {
	// Pace determines which pending pods should be admitted based on the current pod classifications.
	// It returns a subset of NotAdmittedPods that are allowed to proceed.
	Pace(podClassifications PodClassification, logger logr.Logger) (allowPods []corev1.Pod, err error)
	String() string
}

type PacerFactory interface {
	New(key string) Pacer
}
