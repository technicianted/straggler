// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package controller

import (
	"context"
	"fmt"
	"time"

	blockertypes "straggler/pkg/blocker/types"
	"straggler/pkg/controller/types"
	pacertypes "straggler/pkg/pacer/types"

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	DefaultEnableLabel         = "v1.straggler.technicianted/enable"
	DefaultStaggerGroupIDLabel = "v1.straggler.technicianted/group"
	DefaultJobPodLabel         = "v1.straggler.technicianted/jobPod"
	DefaultFlightWait          = 500 * time.Millisecond
)

var _ admission.CustomDefaulter = &Admission{}

// Paces new pod creation using classified pacer.
type Admission struct {
	classifier         types.PodClassifier
	podGroupClassifier types.PodGroupStandingClassifier
	recorderFactory    types.ObjectRecorderFactory
	podBlocker         blockertypes.PodBlocker
	flightTracker      types.AdmissionFlightTracker

	enableLabel         string
	staggerGroupIDLabel string
	jobPodLabel         string

	bypassFailures bool
}

func NewAdmission(classifier types.PodClassifier,
	podGroupClassifier types.PodGroupStandingClassifier,
	recorderFactory types.ObjectRecorderFactory,
	podBlocker blockertypes.PodBlocker,
	flightTracker types.AdmissionFlightTracker,
	bypassFailures bool,
	enableLabel string,
) *Admission {
	return &Admission{
		classifier:          classifier,
		podGroupClassifier:  podGroupClassifier,
		recorderFactory:     recorderFactory,
		podBlocker:          podBlocker,
		flightTracker:       flightTracker,
		enableLabel:         enableLabel,
		staggerGroupIDLabel: DefaultStaggerGroupIDLabel,
		jobPodLabel:         DefaultJobPodLabel,
		bypassFailures:      bypassFailures,
	}
}

func newAdmission(classifier types.PodClassifier,
	podGroupClassifier types.PodGroupStandingClassifier,
	recorderFactory types.ObjectRecorderFactory,
	podBlocker blockertypes.PodBlocker,
	flightTracker types.AdmissionFlightTracker,
	bypassFailures bool,
) *Admission {
	return &Admission{
		classifier:          classifier,
		podGroupClassifier:  podGroupClassifier,
		recorderFactory:     recorderFactory,
		flightTracker:       flightTracker,
		podBlocker:          podBlocker,
		enableLabel:         DefaultEnableLabel,
		staggerGroupIDLabel: DefaultStaggerGroupIDLabel,
		jobPodLabel:         DefaultJobPodLabel,
		bypassFailures:      bypassFailures,
	}
}

func (a *Admission) Default(ctx context.Context, obj runtime.Object) error {
	logger := logf.FromContext(ctx)
	var err error
	switch o := obj.(type) {
	case *corev1.Pod:
		err = a.handlePodAdmission(ctx, o, logger)
	case *batchv1.Job:
		err = a.handleJobAdmission(ctx, o, logger)
	default:
		err = fmt.Errorf("unexpected object type %T", obj)
	}

	if a.bypassFailures && err != nil {
		logger.Info("admission failed, allowing", "error", err)
		err = nil
	}
	return err
}

func (a *Admission) handlePodAdmission(ctx context.Context, pod *corev1.Pod, logger logr.Logger) error {
	logger.V(10).Info("handling admission of pod", "name", pod.Name, "generateName", pod.GenerateName, "namespace", pod.Namespace)
	if !a.checkEnabled(&pod.ObjectMeta, logger) {
		logger.V(0).Info("skipping not enabled pod")
		return nil
	}

	// If this pod belongs to a job with set backoffLimit then we immediately block it
	// since it has to be handled in the reconciler.
	// See job handling for reasonong.
	if len(pod.Labels) > 0 {
		if _, ok := pod.Labels[a.jobPodLabel]; ok {
			return a.blockPod(pod, logger)
		}
	}

	group, err := a.classifier.Classify(pod.ObjectMeta, pod.Spec, logger)
	if err != nil {
		return fmt.Errorf("failed to classify group: %v", err)
	}
	if group == nil {
		logger.Info("pod does not belong to any staggering group")
		return nil
	}
	logger.V(1).Info("staggering group", "id", group.ID, "pacer", group.Pacer)

	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	pod.Labels[a.staggerGroupIDLabel] = group.ID

	logger.V(1).Info("will wait for flight tracker", "wait", DefaultFlightWait)
	flightCTX, cancel := context.WithTimeout(ctx, DefaultFlightWait)
	defer cancel()
	err = a.flightTracker.WaitOne(flightCTX, group.ID, logger)
	if err != nil {
		logger.Info("failed to wait on flight tracker", "error", err)
	}

	ready, starting, blocked, err := a.podGroupClassifier.ClassifyPodGroup(ctx, group.ID, logger)
	if err != nil {
		return fmt.Errorf("failed to classify pod group: %v", err)
	}
	logger.V(1).Info("pod group break down", "ready", len(ready), "starting", len(starting), "blocked", len(blocked))

	unblocked, err := group.Pacer.Pace(pacertypes.PodClassification{
		Ready:    ready,
		Starting: starting,
		// append current pod to blocked and see if it'll be allowed
		Blocked: append(blocked, *pod),
	}, logger)
	if err != nil {
		return fmt.Errorf("failed to pace pod: %v", err)
	}

	defer func() {
		err := a.flightTracker.Track(group.ID, pod.ObjectMeta, logger)
		if err != nil {
			logger.Info("failed to track pod flight", "error", err)
		}
	}()
	for _, unblockedPod := range unblocked {
		if unblockedPod.Name == pod.Name &&
			unblockedPod.Namespace == pod.Namespace &&
			unblockedPod.GenerateName == pod.GenerateName {
			logger.Info("not blocking pod as pacer allows it")
			return nil
		}
	}

	logger.Info("pacer will not allow pod")
	return a.blockPod(pod, logger)
}

func (a *Admission) handleJobAdmission(_ context.Context, job *batchv1.Job, logger logr.Logger) error {
	logger.V(10).Info("handling admission of job", "name", job.Name, "namespace", job.Namespace)
	if !a.checkEnabled(&job.Spec.Template.ObjectMeta, logger) {
		logger.V(0).Info("skipping not enabled job")
		return nil
	}

	// Jobs are tricky controllers becuase of backoffLimit settings.
	// Since pods are immutable, pod unblocking requires deletion of the pod.
	// However by default the Job controller will treat a deleted pod as a failed
	// one and will count against the backoff limit.
	// We need to use job's podFailurePolicy and add onPodConditions DisruptionTarget.
	// Note that this may be a behavior change to the original intent.

	// first check if the policy is enabled
	policyExists := false
	var policyAction batchv1.PodFailurePolicyAction
	if job.Spec.PodFailurePolicy != nil {
		for _, rule := range job.Spec.PodFailurePolicy.Rules {
			policyAction = rule.Action
			for _, condition := range rule.OnPodConditions {
				if condition.Status == corev1.ConditionTrue && condition.Type == corev1.DisruptionTarget {
					policyExists = true
					break
				}
			}
			if policyExists {
				break
			}
		}
	}
	if policyExists && policyAction != batchv1.PodFailurePolicyActionIgnore {
		logger.Info("job already has a defined DisruptionTarget policy and will be bypassed")
		return nil
	}
	if !policyExists {
		logger.Info("patching job to enable pod disruption ignoring")

		if job.Spec.PodFailurePolicy == nil {
			job.Spec.PodFailurePolicy = &batchv1.PodFailurePolicy{}
		}
		job.Spec.PodFailurePolicy.Rules = append(job.Spec.PodFailurePolicy.Rules, batchv1.PodFailurePolicyRule{
			Action: metav1.FieldValidationIgnore,
			OnPodConditions: []batchv1.PodFailurePolicyOnPodConditionsPattern{
				{
					Type:   corev1.DisruptionTarget,
					Status: corev1.ConditionTrue,
				},
			},
		})
	}

	return nil
}

func (a *Admission) checkEnabled(objectMeta *metav1.ObjectMeta, logger logr.Logger) bool {
	if len(objectMeta.Labels) == 0 {
		return false
	}
	if value, ok := objectMeta.Labels[a.enableLabel]; ok {
		if value != "1" {
			logger.Info("found enable label %s but has unexpected value %s", a.enableLabel, value)
			return false
		}
		return true
	}

	return false
}

func (a *Admission) blockPod(pod *corev1.Pod, logger logr.Logger) error {
	logger.V(1).Info("blocking pod", "name", pod.Name, "namespace", pod.Namespace)
	return a.podBlocker.Block(&pod.Spec, logger)
}
