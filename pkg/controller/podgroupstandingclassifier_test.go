// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package controller

import (
	"context"
	"errors"
	"testing"

	blockermocks "straggler/pkg/blocker/mocks"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"straggler/pkg/controller/mocks"
)

func newPodCondition(conditionType corev1.PodConditionType, status corev1.ConditionStatus) corev1.PodCondition {
	return corev1.PodCondition{
		Type:   conditionType,
		Status: status,
	}
}

func TestPodGroupStandingClassifier_ClassifyPodGroup(t *testing.T) {
	groupLabel := DefaultStaggerGroupIDLabel

	ctx := context.TODO()
	logger := logr.Discard()

	tests := []struct {
		name          string
		groupID       string
		podList       []corev1.Pod
		blockedPods   map[string]bool // map of pod name to blocked status
		expectedReady []corev1.Pod
		expectedStart []corev1.Pod
		expectedBlock []corev1.Pod
		listErr       error
	}{
		{
			name:    "Successful Classification with Mixed States",
			groupID: "group1",
			podList: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-ready-1",
						Namespace: "default",
						Labels: map[string]string{
							groupLabel: "group1",
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						Conditions: []corev1.PodCondition{
							newPodCondition(corev1.PodReady, corev1.ConditionTrue),
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-starting-1",
						Namespace: "default",
						Labels: map[string]string{
							groupLabel: "group1",
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						Conditions: []corev1.PodCondition{
							newPodCondition(corev1.PodReady, corev1.ConditionFalse),
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-blocked-1",
						Namespace: "default",
						Labels: map[string]string{
							groupLabel: "group1",
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodPending,
					},
				},
			},
			blockedPods: map[string]bool{
				"pod-blocked-1": true,
			},
			expectedReady: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-ready-1",
						Namespace: "default",
						Labels: map[string]string{
							groupLabel: "group1",
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						Conditions: []corev1.PodCondition{
							newPodCondition(corev1.PodReady, corev1.ConditionTrue),
						},
					},
				},
			},
			expectedStart: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-starting-1",
						Namespace: "default",
						Labels: map[string]string{
							groupLabel: "group1",
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						Conditions: []corev1.PodCondition{
							newPodCondition(corev1.PodReady, corev1.ConditionFalse),
						},
					},
				},
			},
			expectedBlock: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-blocked-1",
						Namespace: "default",
						Labels: map[string]string{
							groupLabel: "group1",
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodPending,
					},
				},
			},
			listErr: nil,
		},
		{
			name:          "No Pods Found",
			groupID:       "group2",
			podList:       []corev1.Pod{},
			blockedPods:   map[string]bool{},
			expectedReady: []corev1.Pod{},
			expectedStart: []corev1.Pod{},
			expectedBlock: []corev1.Pod{},
			listErr:       nil,
		},
		{
			name:          "List Pods Failure",
			groupID:       "group3",
			podList:       nil,
			blockedPods:   map[string]bool{},
			expectedReady: nil,
			expectedStart: nil,
			expectedBlock: nil,
			listErr:       errors.New("failed to list pods"),
		},
		{
			name:    "All Pods Blocked",
			groupID: "group4",
			podList: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-blocked-2",
						Namespace: "default",
						Labels: map[string]string{
							groupLabel: "group4",
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodPending,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-blocked-3",
						Namespace: "default",
						Labels: map[string]string{
							groupLabel: "group4",
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodFailed,
					},
				},
			},
			blockedPods: map[string]bool{
				"pod-blocked-2": true,
				"pod-blocked-3": true,
			},
			expectedReady: []corev1.Pod{},
			expectedStart: []corev1.Pod{},
			expectedBlock: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-blocked-2",
						Namespace: "default",
						Labels: map[string]string{
							groupLabel: "group4",
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodPending,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-blocked-3",
						Namespace: "default",
						Labels: map[string]string{
							groupLabel: "group4",
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodFailed,
					},
				},
			},
			listErr: nil,
		},
		{
			name:    "All Pods Ready",
			groupID: "group5",
			podList: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-ready-2",
						Namespace: "default",
						Labels: map[string]string{
							groupLabel: "group5",
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						Conditions: []corev1.PodCondition{
							newPodCondition(corev1.PodReady, corev1.ConditionTrue),
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-ready-3",
						Namespace: "default",
						Labels: map[string]string{
							groupLabel: "group5",
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						Conditions: []corev1.PodCondition{
							newPodCondition(corev1.PodReady, corev1.ConditionTrue),
						},
					},
				},
			},
			blockedPods: map[string]bool{},
			expectedReady: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-ready-2",
						Namespace: "default",
						Labels: map[string]string{
							groupLabel: "group5",
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						Conditions: []corev1.PodCondition{
							newPodCondition(corev1.PodReady, corev1.ConditionTrue),
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-ready-3",
						Namespace: "default",
						Labels: map[string]string{
							groupLabel: "group5",
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						Conditions: []corev1.PodCondition{
							newPodCondition(corev1.PodReady, corev1.ConditionTrue),
						},
					},
				},
			},
			expectedStart: []corev1.Pod{},
			expectedBlock: []corev1.Pod{},
			listErr:       nil,
		},
	}

	for _, tc := range tests {

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockClient := mocks.NewMockClient(ctrl)
			mockBlocker := blockermocks.NewMockPodBlocker(ctrl)
			classifier := NewPodGroupStandingClassifier(mockClient, mockBlocker)

			// Setup mockClient.List
			if tc.listErr != nil {
				mockClient.
					EXPECT().
					List(
						gomock.Any(), // ctx
						gomock.Any(), // podList
						gomock.Any(), // listOptions...
					).
					Return(tc.listErr)
			} else {
				mockClient.
					EXPECT().
					List(
						gomock.Any(), // ctx
						gomock.Any(), // podList
						gomock.Any(), // listOptions...
					).
					DoAndReturn(func(ctx context.Context, podList *corev1.PodList, opts ...client.ListOption) error {
						podList.Items = tc.podList
						return nil
					})
			}

			// Setup mockBlocker.IsBlocked
			for _, pod := range tc.podList {
				blocked := tc.blockedPods[pod.Name]
				mockBlocker.
					EXPECT().
					IsBlocked(&pod.Spec).
					Return(blocked).
					Times(1)
			}

			// Execute the method under test
			ready, starting, blocked, err := classifier.ClassifyPodGroup(ctx, tc.groupID, logger)

			// Assertions
			if tc.listErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.listErr, err)
				assert.Nil(t, ready)
				assert.Nil(t, starting)
				assert.Nil(t, blocked)
			} else {
				assert.NoError(t, err)
				assert.ElementsMatch(t, tc.expectedReady, ready, "Ready pods do not match")
				assert.ElementsMatch(t, tc.expectedStart, starting, "Starting pods do not match")
				assert.ElementsMatch(t, tc.expectedBlock, blocked, "Blocked pods do not match")
			}
		})
	}
}
