package pacer

import (
	"fmt"
	"stagger/pkg/pacer/types"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
)

var (
	_ types.Pacer = &composite{}
)

type composite struct {
	id     string
	pacers []types.Pacer
}

// Create a composite pacer that allow pods allowed by all pacers.
func NewComposite(id string, pacers []types.Pacer) types.Pacer {
	return &composite{
		id:     id,
		pacers: pacers,
	}
}

func (p *composite) Pace(podClassifications types.PodClassification, logger logr.Logger) (allowPods []corev1.Pod, err error) {
	// find pods that are allowed by _all_ pacers
	results := make(map[string]int)
	for i := range p.pacers {
		result, err := p.pacers[i].Pace(podClassifications, logger)
		if err != nil {
			return nil, err
		}
		for _, pod := range result {
			results[string(pod.UID)] += 1
		}
	}
	// now find pods that were allowed len(pacers) times
	for _, pod := range podClassifications.Blocked {
		if count, ok := results[string(pod.UID)]; ok && count == len(p.pacers) {
			allowPods = append(allowPods, pod)
		}
	}

	return
}

func (p *composite) String() string {
	s := fmt.Sprintf("composite(%s)[%d]:", p.id, len(p.pacers))
	inners := make([]string, 0)
	for _, inner := range p.pacers {
		inners = append(inners, inner.String())
	}
	return s + strings.Join(inners, ",")
}
