// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package controller

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	configtypes "straggler/pkg/config/types"
	"straggler/pkg/controller/types"
	"straggler/pkg/pacer"
	pacertypes "straggler/pkg/pacer/types"

	"github.com/go-logr/logr"
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
	bypassSelector   labels.Selector
}

type groupEntry struct {
	id             string
	configs        []configEntry
	compositePacer pacertypes.Pacer
}

type podClassifier struct {
	sync.Mutex

	configs     map[string]configEntry
	configNames []string
	groupsByID  *cache.Cache
	// TODO: keys can theoritically by deuplicate across configs
	pacersByKey *cache.Cache
}

// Create a new pod classifier into pacer.
func NewPodClassifier() *podClassifier {
	return &podClassifier{
		configs:     make(map[string]configEntry),
		configNames: make([]string, 0),
		groupsByID:  cache.New(30*time.Minute, 1*time.Minute),
		pacersByKey: cache.New(30*time.Minute, 1*time.Minute),
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
	c.configNames = append(c.configNames, entry.Name)

	return nil
}

func (c *podClassifier) RemoveConfig(name string, logger logr.Logger) error {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.configs[name]; !ok {
		return fmt.Errorf("config not found: %s", name)
	}

	delete(c.configs, name)
	newNames := make([]string, 0)
	for _, n := range c.configNames {
		if n != name {
			newNames = append(newNames, n)
		}
	}
	c.configNames = newNames

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
	pacers := make([]pacertypes.Pacer, 0)
	configs := make([]configEntry, 0)
	dummyPod := corev1.Pod{
		ObjectMeta: podMeta,
		Spec:       podSpec,
	}

	for _, name := range c.configNames {
		config := c.configs[name]
		if !config.selector.Matches(labels.Set(dummyPod.Labels)) {
			logger.V(1).Info("skipping config due to label selector", "name", name)
			continue
		}
		if !config.bypassSelector.Empty() &&
			config.bypassSelector.Matches(labels.Set(dummyPod.Labels)) {
			logger.Info("skipping config due to bypass selector match", "name", name)
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
		id := c.calculateGroupID(pacers, configs)
		if g, ok := c.groupsByID.Get(id); ok {
			group = g.(*groupEntry)
		} else {
			group = &groupEntry{
				id:             id,
				configs:        configs,
				compositePacer: pacer.NewComposite(id, pacers),
			}
			c.groupsByID.Set(group.id, group, 0)
		}
	}

	if group != nil {
		return &types.PodClassification{
			ID:    group.id,
			Pacer: group.compositePacer,
		}, nil
	}

	return nil, nil
}

func (c *podClassifier) ClassifyByGroupID(groupID string, logger logr.Logger) (*types.PodClassification, error) {
	g, ok := c.groupsByID.Get(groupID)
	if ok {
		group := g.(*groupEntry)
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
		bypassSelector:   labels.SelectorFromSet(config.BypassLabelSelector),
	}
	return
}

func (c *podClassifier) calculateGroupID(pacers []pacertypes.Pacer, matchedConfigs []configEntry) string {
	configNames := make([]string, 0)
	for _, config := range matchedConfigs {
		configNames = append(configNames, config.Name)
	}
	pacerNames := make([]string, 0)
	for _, pacer := range pacers {
		pacerNames = append(pacerNames, pacer.ID())
	}
	id := fmt.Sprintf("[%s](%s)", strings.Join(pacerNames, ","), strings.Join(configNames, ","))
	hash := md5.New()
	hash.Write([]byte(id))
	return hex.EncodeToString(hash.Sum(nil))
}
