package types

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
)

// PodClassification categorizes pods based on their admission and readiness status.
type PodClassification struct {
	AdmittedAndReadyPods []corev1.Pod
	AdmittedNotReadyPods []corev1.Pod
	NotAdmittedPods      []corev1.Pod
}

//go:generate mockgen -package mocks -destination ../mocks/pacer.go -source $GOFILE

type Pacer interface {
	// Pace determines which pending pods should be admitted based on the current pod classifications.
	// It returns a subset of NotAdmittedPods that are allowed to proceed.
	Pace(podClassifications PodClassification, logger logr.Logger) (allowPods []corev1.Pod, err error)
	String() string
}

type PacerFactory interface {
	New(key string) Pacer
}
