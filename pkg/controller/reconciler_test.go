// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package controller

import (
	"context"
	"errors"
	"testing"

	"straggler/pkg/controller/mocks"
	"straggler/pkg/controller/types"
	pacertypes "straggler/pkg/pacer/types"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	pacermockes "straggler/pkg/pacer/mocks"
)

func setupTest(t *testing.T) (*Reconciler, *mocks.MockClient, *mocks.MockPodClassifier, *mocks.MockPodGroupStandingClassifier, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockClient := mocks.NewMockClient(ctrl)
	mockClassifier := mocks.NewMockPodClassifier(ctrl)
	mockGroupClassifier := mocks.NewMockPodGroupStandingClassifier(ctrl)

	reconciler := NewReconciler(mockClient, mockClassifier, mockGroupClassifier)
	return reconciler, mockClient, mockClassifier, mockGroupClassifier, ctrl
}

func TestReconcile_PodNotFound(t *testing.T) {
	reconciler, mockClient, _, _, ctrl := setupTest(t)
	defer ctrl.Finish()

	req := reconcile.Request{
		NamespacedName: client.ObjectKey{
			Namespace: "default",
			Name:      "nonexistent-pod",
		},
	}

	// Set expectation: Get returns NotFound
	mockClient.
		EXPECT().
		Get(gomock.Any(), req.NamespacedName, gomock.Any()).
		Return(k8serrors.NewNotFound(corev1.Resource("pods"), req.Name)).
		AnyTimes()

	ctx := context.TODO()
	res, err := reconciler.Reconcile(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, res)
}

func TestReconcile_PodNotEnabled(t *testing.T) {
	reconciler, mockClient, _, _, ctrl := setupTest(t)
	defer ctrl.Finish()

	req := reconcile.Request{
		NamespacedName: client.ObjectKey{
			Namespace: "default",
			Name:      "disabled-pod",
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "disabled-pod",
			Labels:    map[string]string{},
		},
	}

	// Set expectation: Get returns the Pod
	mockClient.
		EXPECT().
		Get(
			gomock.Any(),       // ctx
			req.NamespacedName, // key
			gomock.Any(),       // obj
			gomock.Any(),       // opts (variadic)
		).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			*obj.(*corev1.Pod) = *pod
			return nil
		})

	ctx := context.TODO()
	res, err := reconciler.Reconcile(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, res)
}

func TestReconcile_PodClassificationFailure(t *testing.T) {
	reconciler, mockClient, mockClassifier, _, ctrl := setupTest(t)
	defer ctrl.Finish()

	req := reconcile.Request{
		NamespacedName: client.ObjectKey{
			Namespace: "default",
			Name:      "error-pod",
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "error-pod",
			Labels: map[string]string{
				DefaultEnableLabel:         "1",
				DefaultStaggerGroupIDLabel: "groupid",
			},
		},
		Spec: corev1.PodSpec{},
	}

	// Set expectation: Get returns the Pod
	mockClient.
		EXPECT().
		Get(
			gomock.Any(),       // ctx
			req.NamespacedName, // key
			gomock.Any(),       // obj
			gomock.Any(),       // opts (variadic)
		).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			*obj.(*corev1.Pod) = *pod
			return nil
		})

	// Set expectation: Classify returns error
	mockClassifier.
		EXPECT().
		ClassifyByGroupID("groupid", gomock.Any()).
		Return(nil, errors.New("classification error"))

	ctx := context.TODO()
	_, err := reconciler.Reconcile(ctx, req)

	assert.Error(t, err)
}

func TestReconcile_PodNotInAnyGroup(t *testing.T) {
	reconciler, mockClient, mockClassifier, _, ctrl := setupTest(t)
	defer ctrl.Finish()

	req := reconcile.Request{
		NamespacedName: client.ObjectKey{
			Namespace: "default",
			Name:      "ungrouped-pod",
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "ungrouped-pod",
			Labels: map[string]string{
				DefaultEnableLabel:         "1",
				DefaultStaggerGroupIDLabel: "groupid",
			},
		},
		Spec: corev1.PodSpec{},
	}

	// Set expectation: Get returns the Pod
	mockClient.
		EXPECT().
		Get(
			gomock.Any(),       // ctx
			req.NamespacedName, // key
			gomock.Any(),       // obj
			gomock.Any(),       // opts (variadic)
		).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			*obj.(*corev1.Pod) = *pod
			return nil
		})

	// Set expectation: Classify returns nil group
	mockClassifier.
		EXPECT().
		ClassifyByGroupID("groupid", gomock.Any()).
		Return(nil, nil)

	ctx := context.TODO()
	_, err := reconciler.Reconcile(ctx, req)

	assert.Error(t, err)
}

func TestReconcile_SuccessfulReconciliation(t *testing.T) {
	reconciler, mockClient, mockClassifier, mockGroupClassifier, ctrl := setupTest(t)
	defer ctrl.Finish()

	mockPacer := pacermockes.NewMockPacer(ctrl)

	req := reconcile.Request{
		NamespacedName: client.ObjectKey{
			Namespace: "default",
			Name:      "grouped-pod",
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "grouped-pod",
			Labels: map[string]string{
				DefaultEnableLabel:         "1",
				DefaultStaggerGroupIDLabel: "groupid",
			},
		},
		Spec: corev1.PodSpec{},
	}

	group := &types.PodClassification{
		ID:    "groupid",
		Pacer: mockPacer,
	}

	readyPods := []corev1.Pod{
		{ObjectMeta: metav1.ObjectMeta{Name: "ready-pod"}},
	}
	startingPods := []corev1.Pod{
		{ObjectMeta: metav1.ObjectMeta{Name: "starting-pod"}},
	}
	blockedPods := []corev1.Pod{
		*pod,
	}

	// Set expectation: Get returns the Pod
	mockClient.
		EXPECT().
		Get(
			gomock.Any(),       // ctx
			req.NamespacedName, // key
			gomock.Any(),       // obj
			gomock.Any(),       // opts (variadic)
		).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			*obj.(*corev1.Pod) = *pod
			return nil
		})

	// Set expectation: Classify returns the group
	mockClassifier.
		EXPECT().
		ClassifyByGroupID("groupid", gomock.Any()).
		Return(group, nil)

	// Set expectation: ClassifyPodGroup
	mockGroupClassifier.
		EXPECT().
		ClassifyPodGroup(gomock.Any(), "groupid", gomock.Any()).
		Return(readyPods, startingPods, blockedPods, nil)

	// Set expectation: Pacer.Pace
	mockPacer.
		EXPECT().
		Pace(pacertypes.PodClassification{
			Ready:    readyPods,
			Starting: startingPods,
			Blocked:  blockedPods,
		}, gomock.Any()).
		Return([]corev1.Pod{
			{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "blocked-pod"}},
		}, nil)

	mockSubresourceClient := mocks.NewMockSubResourceClient(ctrl)
	mockClient.EXPECT().SubResource("eviction").Return(mockSubresourceClient)
	// Expect Create to be called for eviction
	mockSubresourceClient.
		EXPECT().
		Create(
			gomock.Any(), // ctx
			gomock.Any(), // obj
			gomock.Any(), // subresource
			gomock.Any(), // opts (variadic)
		).
		DoAndReturn(func(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.CreateOption) error {
			pod, ok := obj.(*corev1.Pod)
			assert.True(t, ok)
			assert.Equal(t, "blocked-pod", pod.Name)
			assert.Equal(t, "default", pod.Namespace)
			// Check GracePeriodSeconds
			evict := subResource.(*policyv1.Eviction)
			if evict.DeleteOptions == nil || evict.DeleteOptions.GracePeriodSeconds == nil {
				t.Errorf("GracePeriodSeconds is nil")
			} else {
				assert.Equal(t, int64(0), *evict.DeleteOptions.GracePeriodSeconds)
			}
			return nil
		})

	ctx := context.TODO()
	res, err := reconciler.Reconcile(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, res)
}
