// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package types

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
)

//go:generate mockgen -package mocks -destination ../mocks/blockers.go -source $GOFILE

// PodBlocker is an interface to define functionality for blocking a pod from
// being scheduled until capture jobs are run to completion.
type PodBlocker interface {
	// Block modifies podSpec such that it makes it impossible to schedule.
	Block(podSpec *corev1.PodSpec, logger logr.Logger) error
	// Unblock removes any blocking that was inserted in podSpec.
	Unblock(podSpec *corev1.PodSpec, logger logr.Logger) error
	// IsBlocked checks to see if podSpec has been blocked.
	IsBlocked(podSpec *corev1.PodSpec) bool
}
