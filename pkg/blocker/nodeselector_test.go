package blocker

import (
	"testing"

	"github.com/go-logr/zapr"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

func TestNodeSelectorPodBlockerSimple(t *testing.T) {
	zlog, _ := zap.NewDevelopment()
	logger := zapr.NewLogger(zlog)

	pod := corev1.Pod{}

	b := NewNodeSelectorPodBlocker()
	err := b.Block(&pod.Spec, logger)
	require.NoError(t, err)
	v, ok := pod.Spec.NodeSelector[DefaultNodeSelectorBlockerLabelName]
	require.True(t, ok)
	require.Equal(t, DefaultNodeSelectorBlockerLabelValue, v)

	err = b.Block(&pod.Spec, logger)
	require.NoError(t, err)

	err = b.Unblock(&pod.Spec, logger)
	require.NoError(t, err)
	_, ok = pod.Spec.NodeSelector[DefaultNodeSelectorBlockerLabelName]
	require.False(t, ok)

	err = b.Unblock(&pod.Spec, logger)
	require.NoError(t, err)
}
