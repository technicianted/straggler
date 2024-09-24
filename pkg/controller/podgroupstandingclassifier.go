package controller

import (
	"context"
	"stagger/pkg/controller/types"

	blocker "stagger/pkg/blocker/types"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ types.PodGroupStandingClassifier = &podGroupStandingClassifier{}

type podGroupStandingClassifier struct {
	client     client.Client
	groupLabel string
	blocker    blocker.PodBlocker
}

func NewPodGroupStandingClassifier(client client.Client, blocker blocker.PodBlocker, groupLabel string) types.PodGroupStandingClassifier {
	return &podGroupStandingClassifier{
		client:     client,
		groupLabel: groupLabel,
		blocker:    blocker,
	}
}

func (p *podGroupStandingClassifier) ClassifyPodGroup(ctx context.Context, groupID string, logger logr.Logger) (ready []corev1.Pod, starting []corev1.Pod, blocked []corev1.Pod, err error) {
	logger.Info("classifying pod group", "groupID", groupID)

	podList := &corev1.PodList{}
	listOptions := []client.ListOption{
		client.MatchingLabels{
			p.groupLabel: groupID,
		},
	}

	if err := p.client.List(ctx, podList, listOptions...); err != nil {
		logger.Error(err, "failed to list pods")
		return nil, nil, nil, err
	}

	for _, pod := range podList.Items {
		switch {
		case p.blocker.IsBlocked(&pod.Spec):
			blocked = append(blocked, pod)
		case isPodReady(pod):
			ready = append(ready, pod)
		default:
			starting = append(starting, pod)
		}
	}

	logger.Info("pod group classification complete", "groupID", groupID, "ready", len(ready), "starting", len(starting), "blocked", len(blocked))

	return ready, starting, blocked, nil
}

// Helper function to check if the Pod is Ready
func isPodReady(pod corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
