// Copyright (c) stagger team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package pacer

import (
	"stagger/pkg/pacer/mocks"
	"stagger/pkg/pacer/types"
	"testing"

	"github.com/go-logr/zapr"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCompositePacerSimple(t *testing.T) {
	zlog, _ := zap.NewDevelopment()
	logger := zapr.NewLogger(zlog)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	pendingPods := []corev1.Pod{
		{
			ObjectMeta: v1.ObjectMeta{
				UID: "uid0",
			},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				UID: "uid1",
			},
		},
	}
	// both pacers allow first pod
	pacer1 := mocks.NewMockPacer(mockCtrl)
	pacer1.EXPECT().Pace(gomock.Any(), gomock.Any()).Return([]corev1.Pod{pendingPods[0]}, nil)
	pacer2 := mocks.NewMockPacer(mockCtrl)
	pacer2.EXPECT().Pace(gomock.Any(), gomock.Any()).Return([]corev1.Pod{pendingPods[0]}, nil)

	composite := NewComposite(t.Name(), []types.Pacer{pacer1, pacer2})
	allowedPods, err := composite.Pace(types.PodClassification{
		Blocked: pendingPods,
	}, logger)
	require.NoError(t, err)
	require.Len(t, allowedPods, 1)
	require.EqualValues(t, pendingPods[0:1], allowedPods)

	// pacers allow different pods
	pacer1.EXPECT().Pace(gomock.Any(), gomock.Any()).Return([]corev1.Pod{pendingPods[0]}, nil)
	pacer2.EXPECT().Pace(gomock.Any(), gomock.Any()).Return([]corev1.Pod{pendingPods[1]}, nil)
	allowedPods, err = composite.Pace(types.PodClassification{
		Blocked: pendingPods,
	}, logger)
	require.NoError(t, err)
	require.Len(t, allowedPods, 0)
}
