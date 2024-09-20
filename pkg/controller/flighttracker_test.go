package controller

import (
	"context"
	"stagger/pkg/controller/mocks"
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

	err := tracker.Track(groupingKey, podMeta, logger)
	require.NoError(t, err)

	// this should timeout since flight hasn't landed
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err = tracker.WaitOne(ctx, groupingKey, logger)
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
	err = tracker.WaitOne(ctx, groupingKey, logger)
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

	err := tracker.Track(groupingKey, podMeta, logger)
	require.NoError(t, err)

	// this should timeout since flight hasn't landed
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err = tracker.WaitOne(ctx, groupingKey, logger)
	require.Error(t, err)
	require.Equal(t, context.DeadlineExceeded, err)

	// wait for auto landing
	time.Sleep(300 * time.Millisecond)

	// this should not timeout because of autolanding
	ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err = tracker.WaitOne(ctx, groupingKey, logger)
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
	//pod2 := corev1.Pod{
	//	ObjectMeta: pod2Meta,
	//}

	mockClient := mocks.NewMockClient(mockCtrl)
	mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, object apitypes.NamespacedName, obj client.Object, _ ...client.GetOption) error {
			require.EqualValues(t, object.Name, pod1.Name)
			newPod := obj.(*corev1.Pod)
			pod1.DeepCopyInto(newPod)
			return nil
		})
	tracker := NewFlightTracker(mockClient, time.Second, keyLabel, logger)

	err := tracker.Track(groupingKey, pod1Meta, logger)
	require.NoError(t, err)

	// land
	pod1.Name = pod1.GenerateName + "1"
	_, err = tracker.Reconcile(
		context.Background(),
		reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(&pod1),
		})
	require.NoError(t, err)

	// this should not timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err = tracker.WaitOne(ctx, groupingKey, logger)
	require.NoError(t, err)

	err = tracker.Track(groupingKey, pod2Meta, logger)
	require.NoError(t, err)
	// reconcile pod1 again, should be a noop
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

	// should timeout since pod2 hasn't landed
	ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err = tracker.WaitOne(ctx, groupingKey, logger)
	require.Error(t, err)
	require.Equal(t, context.DeadlineExceeded, err)
}
