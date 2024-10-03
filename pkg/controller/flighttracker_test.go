// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package controller

import (
	"context"
	"straggler/pkg/controller/mocks"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestFlightTrackerSimple(t *testing.T) {
	zlog, _ := zap.NewDevelopment()
	logger := zapr.NewLogger(zlog)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	groupingKey := "key1"
	keyLabel := "key"
	podMeta := metav1.ObjectMeta{
		Name: "pod1",
		Labels: map[string]string{
			keyLabel: groupingKey,
		},
	}
	pod := corev1.Pod{
		ObjectMeta: podMeta,
	}

	mockClient := mocks.NewMockClient(mockCtrl)
	mockClient.EXPECT().Get(gomock.Any(), apitypes.NamespacedName{Name: pod.Name}, gomock.Any()).DoAndReturn(
		func(_ context.Context, object apitypes.NamespacedName, obj client.Object, _ ...client.GetOption) error {
			require.EqualValues(t, object.Name, pod.Name)
			newPod := obj.(*corev1.Pod)
			pod.DeepCopyInto(newPod)
			return nil
		})
	tracker := NewFlightTracker(mockClient, time.Second, keyLabel, logger)
	// this should timeout since there are no previous flights
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := tracker.WaitOne(ctx, groupingKey, podMeta, logger)
	require.NoError(t, err)

	// timeout since flight hasn't landed
	ctx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err = tracker.WaitOne(ctx, groupingKey, podMeta, logger)
	require.Error(t, err)
	require.Equal(t, context.DeadlineExceeded, err)

	// land
	_, err = tracker.Reconcile(
		context.Background(),
		reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(&pod),
		})
	require.NoError(t, err)

	// this should not timeout
	ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err = tracker.WaitOne(ctx, groupingKey, podMeta, logger)
	require.NoError(t, err)
}

func TestFlightTrackerAutoLanding(t *testing.T) {
	zlog, _ := zap.NewDevelopment()
	logger := zapr.NewLogger(zlog)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	groupingKey := "key1"
	keyLabel := "key"
	podMeta := metav1.ObjectMeta{
		Name: "pod1",
		Labels: map[string]string{
			keyLabel: groupingKey,
		},
	}

	mockClient := mocks.NewMockClient(mockCtrl)
	tracker := NewFlightTracker(mockClient, 200*time.Millisecond, keyLabel, logger)

	// no timeout since it is first flight
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := tracker.WaitOne(ctx, groupingKey, podMeta, logger)
	require.NoError(t, err)

	// timeout since flight hasn't landed
	ctx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err = tracker.WaitOne(ctx, groupingKey, podMeta, logger)
	require.Error(t, err)
	require.Equal(t, context.DeadlineExceeded, err)

	// wait for auto landing
	time.Sleep(300 * time.Millisecond)

	// this should not timeout because of autolanding
	ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err = tracker.WaitOne(ctx, groupingKey, podMeta, logger)
	require.NoError(t, err)
}

func TestFlightTrackerSeenPods(t *testing.T) {
	zlog, _ := zap.NewDevelopment()
	logger := zapr.NewLogger(zlog)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	groupingKey := "key1"
	keyLabel := "key"
	pod1Meta := metav1.ObjectMeta{
		GenerateName: "pod",
		Labels: map[string]string{
			keyLabel: groupingKey,
		},
	}
	pod1 := corev1.Pod{
		ObjectMeta: pod1Meta,
	}
	pod2Meta := metav1.ObjectMeta{
		GenerateName: "pod",
		Labels: map[string]string{
			keyLabel: groupingKey,
		},
	}

	mockClient := mocks.NewMockClient(mockCtrl)

	tracker := NewFlightTracker(mockClient, time.Second, keyLabel, logger)

	// this should not timeout since it's the first one
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := tracker.WaitOne(ctx, groupingKey, pod1Meta, logger)
	require.NoError(t, err)

	// land pod1
	mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, object apitypes.NamespacedName, obj client.Object, _ ...client.GetOption) error {
			require.EqualValues(t, object.Name, pod1.Name)
			newPod := obj.(*corev1.Pod)
			pod1.DeepCopyInto(newPod)
			return nil
		})
	_, err = tracker.Reconcile(
		context.Background(),
		reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(&pod1),
		})
	require.NoError(t, err)
	// track pod2. should not timeout
	err = tracker.WaitOne(ctx, groupingKey, pod2Meta, logger)
	require.NoError(t, err)
	// land pod1 again, should be a noop
	mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, object apitypes.NamespacedName, obj client.Object, _ ...client.GetOption) error {
			require.EqualValues(t, object.Name, pod1.Name)
			newPod := obj.(*corev1.Pod)
			pod1.DeepCopyInto(newPod)
			return nil
		})
	_, err = tracker.Reconcile(
		context.Background(),
		reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(&pod1),
		})
	require.NoError(t, err)

	// start pod1 again but do not land it, should not timeout
	// since there are no in-flight pods.
	ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err = tracker.WaitOne(ctx, groupingKey, pod1Meta, logger)
	require.NoError(t, err)
	// should timeout since pod1 hasn't landed
	ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err = tracker.WaitOne(ctx, groupingKey, pod2Meta, logger)
	require.Error(t, err)
	require.Equal(t, context.DeadlineExceeded, err)
}
