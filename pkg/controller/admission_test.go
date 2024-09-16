package controller

import (
	"context"
	"fmt"
	"testing"

	blockermocks "stagger/pkg/blocker/mocks"
	"stagger/pkg/controller/mocks"
	"stagger/pkg/controller/types"
	pacermocks "stagger/pkg/pacer/mocks"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAdmissionEnableLabel(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	pod := corev1.Pod{}

	classifier := mocks.NewMockPodClassifier(mockCtrl)
	podGroupClassifier := mocks.NewMockPodGroupStandingClassifier(mockCtrl)
	recorderFactory := mocks.NewMockObjectRecorderFactory(mockCtrl)
	blocker := blockermocks.NewMockPodBlocker(mockCtrl)

	admission := newAdmission(classifier, podGroupClassifier, recorderFactory, blocker, false)
	err := admission.Default(context.Background(), &pod)
	require.NoError(t, err)
	require.EqualValues(t, corev1.Pod{}, pod)
}

func TestAdmissionPodAdmissionBlocking(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	pod := corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Labels: map[string]string{
				DefaultEnableLabel: "1",
			},
		},
	}

	pacer := pacermocks.NewMockPacer(mockCtrl)
	// do no allow any pods
	pacer.EXPECT().Pace(gomock.Any(), gomock.Any()).Return(nil, nil)

	classifier := mocks.NewMockPodClassifier(mockCtrl)
	classifier.EXPECT().Classify(pod.ObjectMeta, pod.Spec, gomock.Any()).Return(&types.PodClassification{
		ID:    "testid",
		Pacer: pacer,
	}, nil)
	podGroupClassifier := mocks.NewMockPodGroupStandingClassifier(mockCtrl)
	podGroupClassifier.EXPECT().ClassifyPodGroup(gomock.Any(), "testid", gomock.Any()).Return(nil, nil, nil, nil).Times(2)
	recorderFactory := mocks.NewMockObjectRecorderFactory(mockCtrl)
	blocker := blockermocks.NewMockPodBlocker(mockCtrl)
	blocker.EXPECT().Block(gomock.Any(), gomock.Any()).Return(nil)

	admission := newAdmission(classifier, podGroupClassifier, recorderFactory, blocker, false)
	err := admission.Default(context.Background(), &pod)
	require.NoError(t, err)
	// check group label
	require.Equal(t, "testid", pod.Labels[DefaultStaggerGroupIDLabel])

	// allow pod. we expect the group label but not blocking
	pod = corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Labels: map[string]string{
				DefaultEnableLabel: "1",
			},
		},
	}
	pacer.EXPECT().Pace(gomock.Any(), gomock.Any()).Return([]corev1.Pod{pod}, nil)
	classifier.EXPECT().Classify(pod.ObjectMeta, pod.Spec, gomock.Any()).Return(&types.PodClassification{
		ID:    "testid",
		Pacer: pacer,
	}, nil)
	err = admission.Default(context.Background(), &pod)
	require.NoError(t, err)
	// check group label
	require.Equal(t, "testid", pod.Labels[DefaultStaggerGroupIDLabel])

	// test classifier returning nil group
	pod = corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Labels: map[string]string{
				DefaultEnableLabel: "1",
			},
		},
	}
	classifier.EXPECT().Classify(pod.ObjectMeta, pod.Spec, gomock.Any()).Return(nil, nil)
	err = admission.Default(context.Background(), &pod)
	require.NoError(t, err)
	// check group label
	require.NotContains(t, DefaultStaggerGroupIDLabel, pod.Labels)
}

func TestAdmissionPodErrorBypass(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	pod := corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Labels: map[string]string{
				DefaultEnableLabel: "1",
			},
		},
	}

	classifier := mocks.NewMockPodClassifier(mockCtrl)
	classifier.EXPECT().Classify(pod.ObjectMeta, pod.Spec, gomock.Any()).Return(nil, fmt.Errorf("test error")).Times(2)
	podGroupClassifier := mocks.NewMockPodGroupStandingClassifier(mockCtrl)
	recorderFactory := mocks.NewMockObjectRecorderFactory(mockCtrl)
	blocker := blockermocks.NewMockPodBlocker(mockCtrl)

	// we should get an error
	admission := newAdmission(classifier, podGroupClassifier, recorderFactory, blocker, false)
	err := admission.Default(context.Background(), &pod)
	require.Error(t, err)

	// we should not get an error
	admission = newAdmission(classifier, podGroupClassifier, recorderFactory, blocker, true)
	err = admission.Default(context.Background(), &pod)
	require.NoError(t, err)
}

func TestAdmissionJobSimple(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	classifier := mocks.NewMockPodClassifier(mockCtrl)
	podGroupClassifier := mocks.NewMockPodGroupStandingClassifier(mockCtrl)
	recorderFactory := mocks.NewMockObjectRecorderFactory(mockCtrl)
	blocker := blockermocks.NewMockPodBlocker(mockCtrl)

	job := batchv1.Job{
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						DefaultEnableLabel: "1",
					},
				},
			},
		},
	}
	admission := newAdmission(classifier, podGroupClassifier, recorderFactory, blocker, false)
	err := admission.Default(context.Background(), &job)
	require.NoError(t, err)
	// check if policy was added
	require.NotNil(t, job.Spec.PodFailurePolicy)
	require.Len(t, job.Spec.PodFailurePolicy.Rules, 1)

	// skip adding if exists
	err = admission.Default(context.Background(), &job)
	require.NoError(t, err)
	// should still be 1
	require.NotNil(t, job.Spec.PodFailurePolicy)
	require.Len(t, job.Spec.PodFailurePolicy.Rules, 1)
}
