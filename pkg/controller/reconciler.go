package controller

import (
	"context"
	"fmt"
	"stagger/pkg/controller/types"
	pacertypes "stagger/pkg/pacer/types"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Continuously monitor pod changes and make sure that pacers
// are updated.
type Reconciler struct {
	client             client.Client
	classifier         types.PodClassifier
	podGroupClassifier types.PodGroupStandingClassifier

	enableLabel         string
	staggerGroupIDLabel string
}

var _ reconcile.Reconciler = &Reconciler{}

func NewReconciler(client client.Client, classifier types.PodClassifier, podGroupClassifier types.PodGroupStandingClassifier) *Reconciler {
	return &Reconciler{
		client:             client,
		classifier:         classifier,
		podGroupClassifier: podGroupClassifier,

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
			logger.Error(err, "Failed to get Pod")
			return reconcile.Result{}, err
		}
		// Pod not found; it might have been deleted after the reconcile request.
		logger.Info("Pod not found; it might have been deleted", "pod", request.NamespacedName)
		return reconcile.Result{}, nil
	}

	if !r.checkEnabled(&pod.ObjectMeta, logger) {
		logger.V(10).Info("skipping not enabled pod")
		return reconcile.Result{}, nil
	}

	group, err := r.classifier.Classify(pod.ObjectMeta, pod.Spec, logger)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to classify group: %v", err)
	}
	if group == nil {
		logger.Info("pod does not belong to any staggering group")
		return reconcile.Result{}, nil
	}
	logger.V(1).Info("staggering group", "id", group.ID, "pacer", group.Pacer)

	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	pod.Labels[r.staggerGroupIDLabel] = group.ID

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

	// evict all the unblocked pods
	for _, unblockedPod := range unblocked {
		if err := evictPod(ctx, r.client, &unblockedPod); err != nil {
			logger.Error(err, "failed to evict pod", "pod", unblockedPod.Name, "namespace", unblockedPod.Namespace)
		}
	}

	return reconcile.Result{}, nil
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
