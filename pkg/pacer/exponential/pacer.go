package exponential

import (
	"fmt"
	"stagger/pkg/pacer/types"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
)

var (
	_ types.Pacer = &pacer{}
)

type pacer struct {
	name   string
	key    string
	config Config
}

func New(name string, key string, config Config) *pacer {
	return &pacer{
		name:   name,
		key:    key,
		config: config,
	}
}

// Pace determines which pods are allowed to be admitted based on exponential pacing.
func (p *pacer) Pace(podClassifications types.PodClassification, logger logr.Logger) (allowPods []corev1.Pod, err error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *pacer) String() string {
	return fmt.Sprintf("%T[%s]: %s", p, p.name, p.key)
}
