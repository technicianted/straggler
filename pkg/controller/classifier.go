package controller

import (
	"fmt"
	"sync"
	"time"

	configtypes "stagger/pkg/config/types"
	"stagger/pkg/controller/types"
	"stagger/pkg/pacer"
	pacertypes "stagger/pkg/pacer/types"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/ohler55/ojg/jp"
	"github.com/patrickmn/go-cache"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var (
	_ types.PodClassifier             = &podClassifier{}
	_ types.PodClassifierConfigurator = &podClassifier{}
)

type configEntry struct {
	configtypes.StaggerGroup

	groupingJSONPath jp.Expr
	selector         labels.Selector
}

type groupEntry struct {
	id             string
	configs        []configEntry
	compositePacer pacertypes.Pacer
}

type podClassifier struct {
	sync.Mutex

	configs        map[string]configEntry
	groupsByPodUID *cache.Cache
	groupsByID     *cache.Cache
	// TODO: keys can theoritically by deuplicate across configs
	pacersByKey *cache.Cache
}

// Create a new pod classifier into pacer.
func NewPodClassifier() *podClassifier {
	return &podClassifier{
		configs:        make(map[string]configEntry),
		groupsByPodUID: cache.New(30*time.Minute, 1*time.Minute),
		groupsByID:     cache.New(30*time.Minute, 1*time.Minute),
		pacersByKey:    cache.New(30*time.Minute, 1*time.Minute),
	}
}

func (c *podClassifier) AddConfig(config configtypes.StaggerGroup, logger logr.Logger) error {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.configs[config.Name]; ok {
		return fmt.Errorf("duplicate config name: %s", config.Name)
	}

	entry, err := c.newConfigEntryLocked(config)
	if err != nil {
		return err
	}
	c.configs[entry.Name] = entry

	return nil
}

func (c *podClassifier) RemoveConfig(name string, logger logr.Logger) error {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.configs[name]; !ok {
		return fmt.Errorf("config not found: %s", name)
	}

	delete(c.configs, name)

	return nil
}

func (c *podClassifier) UpdateConfig(config configtypes.StaggerGroup, logger logr.Logger) error {
	c.Lock()
	defer c.Unlock()

	entry, err := c.newConfigEntryLocked(config)
	if err != nil {
		return err
	}

	delete(c.configs, config.Name)
	c.configs[entry.Name] = entry

	return nil
}

func (c *podClassifier) Classify(podMeta metav1.ObjectMeta, podSpec corev1.PodSpec, logger logr.Logger) (*types.PodClassification, error) {
	logger.V(10).Info("classifying pod", "name", podMeta.Name, "namespace", podMeta.Name, "uid", podMeta.UID)

	c.Lock()
	defer c.Unlock()

	var group *groupEntry
	item, ok := c.groupsByPodUID.Get(string(podMeta.UID))
	if ok {
		group = item.(*groupEntry)
	} else {
		pacers := make([]pacertypes.Pacer, 0)
		configs := make([]configEntry, 0)
		dummyPod := corev1.Pod{
			ObjectMeta: podMeta,
			Spec:       podSpec,
		}

		for name := range c.configs {
			config := c.configs[name]
			if !config.selector.Matches(labels.Set(dummyPod.Labels)) {
				logger.V(1).Info("skipping config due to label selector", "name", name)
				continue
			}

			results := config.groupingJSONPath.Get(dummyPod)
			if len(results) == 0 {
				logger.V(1).Info("skipping config due to empty json path selector", "name", name)
				continue
			}
			key := ""
			for _, result := range results {
				key += fmt.Sprintf("%v", result)
			}
			if len(key) == 0 {
				logger.V(1).Info("skipping config due to empty json path selector", "name", name)
				continue
			}
			logger.V(10).Info("obtained grouping key", "key", key, "jsonpathResults", len(results))

			var pacer pacertypes.Pacer
			pacerItem, ok := c.pacersByKey.Get(key)
			if !ok {
				pacer = config.PacerFactory.New(key)
			} else {
				pacer = pacerItem.(pacertypes.Pacer)
			}
			c.pacersByKey.Set(key, pacer, 0)
			pacers = append(pacers, pacer)
			configs = append(configs, config)
		}

		if len(configs) > 0 {
			id := uuid.New().String()
			group = &groupEntry{
				id:             id,
				configs:        configs,
				compositePacer: pacer.NewComposite(id, pacers),
			}
			c.groupsByID.Set(group.id, group, 0)
		}
	}

	if group != nil {
		c.groupsByPodUID.Set(string(podMeta.UID), group, 0)

		return &types.PodClassification{
			ID:    group.id,
			Pacer: group.compositePacer,
		}, nil
	}

	return nil, nil
}

func (c *podClassifier) newConfigEntryLocked(config configtypes.StaggerGroup) (entry configEntry, err error) {
	if len(config.GroupingExpression) == 0 {
		err = fmt.Errorf("empty grouping expression")
		return
	}
	// no duplicate expressions.
	for name := range c.configs {
		if config.GroupingExpression == c.configs[name].GroupingExpression {
			err = fmt.Errorf("grouping expression already exists: %s", name)
			return
		}
	}
	expr, err := jp.ParseString(config.GroupingExpression)
	if err != nil {
		err = fmt.Errorf("failed to parse jsonpath %s: %v", config.GroupingExpression, err)
		return
	}

	entry = configEntry{
		StaggerGroup:     config,
		groupingJSONPath: expr,
		selector:         labels.SelectorFromSet(config.LabelSelector),
	}
	return
}
