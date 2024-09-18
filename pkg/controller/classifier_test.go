package controller

import (
	"stagger/pkg/config/types"
	"stagger/pkg/pacer/mocks"
	"testing"

	"github.com/go-logr/zapr"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClassifierClassifySuccess(t *testing.T) {
	zlog, _ := zap.NewDevelopment()
	logger := zapr.NewLogger(zlog)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	testNamespace := "testnamespace"
	pacer := mocks.NewMockPacer(mockCtrl)
	pacer.EXPECT().ID().Return("pacer").AnyTimes()
	pacerFactory := mocks.NewMockPacerFactory(mockCtrl)
	pacerFactory.EXPECT().New(testNamespace).Return(pacer)

	classifier := NewPodClassifier()
	err := classifier.AddConfig(types.StaggerGroup{
		GroupingExpression: ".metadata.namespace",
		PacerFactory:       pacerFactory,
	}, logger)
	require.NoError(t, err)

	pod := corev1.Pod{
		ObjectMeta: v1.ObjectMeta{Namespace: testNamespace},
	}
	result, err := classifier.Classify(pod.ObjectMeta, pod.Spec, logger)
	require.NoError(t, err)
	require.NotNil(t, result)

	// another classification should yield same cached results
	result, err = classifier.Classify(pod.ObjectMeta, pod.Spec, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestClassifierClassifyMultiSuccess(t *testing.T) {
	zlog, _ := zap.NewDevelopment()
	logger := zapr.NewLogger(zlog)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	testNamespace := "testnamespace"
	testLabelName := "label1"
	testLabelvalue := "value1"
	// this factory coalled for config1 with namespace matching
	pacer1 := mocks.NewMockPacer(mockCtrl)
	pacer1.EXPECT().ID().Return("pacer1").AnyTimes()
	pacerFactory1 := mocks.NewMockPacerFactory(mockCtrl)
	pacerFactory1.EXPECT().New(testNamespace).Return(pacer1)
	// this factory called for config2 with label matching
	pacer2 := mocks.NewMockPacer(mockCtrl)
	pacer2.EXPECT().ID().Return("pacer2").AnyTimes()
	pacerFactory2 := mocks.NewMockPacerFactory(mockCtrl)
	pacerFactory2.EXPECT().New(testLabelvalue).Return(pacer2)

	classifier := NewPodClassifier()
	err := classifier.AddConfig(types.StaggerGroup{
		Name:               "config1",
		GroupingExpression: ".metadata.namespace",
		PacerFactory:       pacerFactory1,
	}, logger)
	require.NoError(t, err)
	err = classifier.AddConfig(types.StaggerGroup{
		Name:               "config2",
		GroupingExpression: ".metadata.labels." + testLabelName,
		PacerFactory:       pacerFactory2,
	}, logger)
	require.NoError(t, err)

	pod := corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Namespace: testNamespace,
			Labels: map[string]string{
				testLabelName: testLabelvalue,
			},
		},
	}
	result, err := classifier.Classify(pod.ObjectMeta, pod.Spec, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestClassifierSkipSelector(t *testing.T) {
	zlog, _ := zap.NewDevelopment()
	logger := zapr.NewLogger(zlog)

	classifier := NewPodClassifier()
	err := classifier.AddConfig(types.StaggerGroup{
		LabelSelector:      map[string]string{"key": "value"},
		GroupingExpression: ".metadata.name",
	}, logger)
	require.NoError(t, err)

	pod := corev1.Pod{}
	result, err := classifier.Classify(pod.ObjectMeta, pod.Spec, logger)
	require.NoError(t, err)
	require.Nil(t, result)
}

func TestClassifierSkipNoKey(t *testing.T) {
	zlog, _ := zap.NewDevelopment()
	logger := zapr.NewLogger(zlog)

	classifier := NewPodClassifier()
	err := classifier.AddConfig(types.StaggerGroup{
		// this won't match
		GroupingExpression: ".metadata.name",
	}, logger)
	require.NoError(t, err)

	pod := corev1.Pod{}
	result, err := classifier.Classify(pod.ObjectMeta, pod.Spec, logger)
	require.NoError(t, err)
	require.Nil(t, result)
}

func TestClassifierBadJSONPath(t *testing.T) {
	zlog, _ := zap.NewDevelopment()
	logger := zapr.NewLogger(zlog)

	classifier := NewPodClassifier()
	err := classifier.AddConfig(types.StaggerGroup{
		GroupingExpression: "bad jsonpath",
	}, logger)
	require.Error(t, err)
}
