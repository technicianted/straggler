package types

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
)

//go:generate mockgen -package mocks -destination ../mocks/pacer.go -source $GOFILE

type Pacer interface {
	// Give set of readyPods, which of pendingPods should be allowed to proceed.
	Pace(readyPods []corev1.Pod, pendingPods []corev1.Pod, logger logr.Logger) (allowPods []corev1.Pod, err error)
	String() string
}

type PacerFactory interface {
	New(key string) Pacer
}
