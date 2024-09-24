// Copyright (c) stagger team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package controller

import (
	"context"

	"stagger/pkg/controller/types"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ types.AdmissionFlightTracker = &noopFlightTracker{}

type noopFlightTracker struct{}

func (n *noopFlightTracker) Track(key string, object metav1.ObjectMeta, logger logr.Logger) error {
	return nil
}

func (n *noopFlightTracker) WaitOne(ctx context.Context, key string, logger logr.Logger) error {
	return nil
}
