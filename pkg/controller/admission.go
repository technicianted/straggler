package controller

import (
	"context"

	"stagger/pkg/controller/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var DefaultStaggerGroupLabel = "v1.stagger.technicianted/group"

// Paces new pod creation using classified pacer.
type Admission struct {
	client          client.Client
	classifier      *podClassifier
	recorderFactory types.ObjectRecorderFactory
}

var _ admission.Handler = &Admission{}

func (a *Admission) Handle(ctx context.Context, req admission.Request) admission.Response {
	// classify
	// get pods in stagger group
	// set StaggerGroupLabel
	// pace -> block if needed
	return admission.Response{}
}
