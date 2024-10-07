// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package controller

import (
	"context"
	"fmt"
	"math"
	"time"

	"straggler/pkg/controller/types"
	pacertypes "straggler/pkg/pacer/types"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	DefaultBlockedPodResyncDuration = 1 * time.Minute
)

// Continuously monitor pod changes and make sure that pacers
// are updated.
type Reconciler struct {
	client                   client.Client
	classifier               types.PodClassifier
	podGroupClassifier       types.PodGroupStandingClassifier
	blockedPodResyncDuration time.Duration

	enableLabel         string
	staggerGroupIDLabel string
}

var _ reconcile.Reconciler = &Reconciler{}

func NewReconciler(client client.Client, classifier types.PodClassifier, podGroupClassifier types.PodGroupStandingClassifier) *Reconciler {
	return &Reconciler{
		client:                   client,
		classifier:               classifier,
		podGroupClassifier:       podGroupClassifier,
		blockedPodResyncDuration: DefaultBlockedPodResyncDuration,

		enableLabel:         DefaultEnableLabel,
		staggerGroupIDLabel: DefaultStaggerGroupIDLabel,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	// Fetch the Pod instance being reconciled
	logger := logf.FromContext(ctx)
	pod := &corev1.Pod{}
	err := r.client.Get(ctx, request.NamespacedName, pod)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "failed to get Pod")
			return reconcile.Result{}, err
		}
		// Pod not found; it might have been deleted after the reconcile request.
		logger.Info("pod not found; it might have been deleted", "pod", request.NamespacedName)
		return reconcile.Result{}, nil
	}

	if !r.checkEnabled(&pod.ObjectMeta, logger) {
		logger.V(10).Info("skipping not enabled pod")
		return reconcile.Result{}, nil
	}

	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	if _, ok := pod.Labels[DefaultStaggeredPodLabel]; !ok {
		logger.V(1).Info("pod is not staggered")
		return reconcile.Result{}, nil
	}
	groupID, ok := pod.Labels[r.staggerGroupIDLabel]
	if !ok {
		logger.Info("pod does not have group ID label")
		return reconcile.Result{}, nil
	}
	if len(groupID) == 0 {
		logger.V(1).Info("pod has nil group ID")
		return reconcile.Result{}, nil
	}

	group, err := r.classifier.ClassifyByGroupID(groupID, logger)
	if err != nil {
		return reconcile.Result{}, err
	}
	if group == nil {
		return reconcile.Result{}, fmt.Errorf("pod group ID not found: %v", groupID)
	}

	logger.V(1).Info("staggering group", "id", group.ID, "pacer", group.Pacer)

	ready, starting, blocked, err := r.podGroupClassifier.ClassifyPodGroup(ctx, group.ID, logger)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to classify pod group: %v", err)
	}
	logger.V(1).Info("pod group break down", "ready", len(ready), "starting", len(starting), "blocked", len(blocked))

	unblocked, err := group.Pacer.Pace(pacertypes.PodClassification{
		Ready:    ready,
		Starting: starting,
		Blocked:  blocked,
	}, logger)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to pace pod: %v", err)
	}

	unblockedPods := map[apitypes.NamespacedName]bool{}
	// evict all the unblocked pods
	for _, unblockedPod := range unblocked {
		logger.V(1).Info("evicting pod to unblock it")
		if err := evictPod(ctx, r.client, &unblockedPod); client.IgnoreNotFound(err) != nil {
			logger.Error(err, "failed to evict pod", "pod", unblockedPod.Name, "namespace", unblockedPod.Namespace)
		} else {
			unblockedPods[client.ObjectKeyFromObject(&unblockedPod)] = true
		}
	}

	if _, ok := unblockedPods[request.NamespacedName]; ok {
		return reconcile.Result{}, nil
	}

	// if our pod wasn't unblocked then check for policies.
	policyMaxDuration := group.GroupPolicies.MaxBlockedDuration
	// apply max blocked policy.
	var durationUntilUnblock time.Duration
	if policyMaxDuration > 0 && !pod.CreationTimestamp.IsZero() {
		timeSinceCreation := time.Since(pod.CreationTimestamp.Time)
		logger.V(1).Info("checking MaxBlockedDuration", "maxDuration", policyMaxDuration, "creationDuration", timeSinceCreation)
		durationUntilUnblock = policyMaxDuration - timeSinceCreation
		if durationUntilUnblock <= 0 {
			logger.Info("blocked pod exceeded policy duration", "maxDuration", policyMaxDuration)
			if err := evictPod(ctx, r.client, pod); client.IgnoreNotFound(err) != nil {
				logger.Error(err, "failed to evict pod", "pod", pod.Name, "namespace", pod.Namespace)
			} else {
				// success, return default
				return reconcile.Result{}, nil
			}
		}
	}

	// if a max blocking time specified then we need to resync based on that.
	resync := r.blockedPodResyncDuration
	if group.GroupPolicies.MaxBlockedDuration > 0 && durationUntilUnblock > 0 {
		resync = time.Duration(math.Min(
			float64(resync),
			float64(durationUntilUnblock)))
	}
	return reconcile.Result{
		RequeueAfter: resync,
	}, nil

}

func (r *Reconciler) checkEnabled(objectMeta *metav1.ObjectMeta, logger logr.Logger) bool {
	if len(objectMeta.Labels) == 0 {
		return false
	}
	if value, ok := objectMeta.Labels[r.enableLabel]; ok {
		if value != "1" {
			logger.Info("found enable label %s but has unexpected value %s", r.enableLabel, value)
			return false
		}
		return true
	}

	return false
}

func evictPod(ctx context.Context, cl client.Client, pod *corev1.Pod) error {
	eviction := &policyv1.Eviction{
		DeleteOptions: &metav1.DeleteOptions{GracePeriodSeconds: ptr.To(int64(0))},
	}

	return cl.SubResource("eviction").Create(ctx, pod, eviction, &client.SubResourceCreateOptions{})
}
