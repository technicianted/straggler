package controller

import (
	"context"
	"fmt"
	"stagger/pkg/controller/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Continuously monitor pod changes and make sure that pacers
// are updated.
type Reconciler struct {
	client          client.Client
	classifier      *podClassifier
	recorderFactory types.ObjectRecorderFactory
}

var _ reconcile.Reconciler = &Reconciler{}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	// get StaggerGroup label
	// get pods matching label
	// pace -> delete unblocked pods
	return reconcile.Result{}, fmt.Errorf("not implemented")
}
