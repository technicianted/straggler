// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package controller

import (
	"context"

	"straggler/pkg/controller/types"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ types.AdmissionFlightTracker = &noopFlightTracker{}

type noopFlightTracker struct{}

func (n *noopFlightTracker) WaitOne(ctx context.Context, key string, object metav1.ObjectMeta, logger logr.Logger) error {
	return nil
}
