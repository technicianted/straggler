// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package blocker

import (
	"straggler/pkg/blocker/types"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
)

var (
	DefaultNodeSelectorBlockerLabelName  = "v1.straggler.technicianted/doNotSchedule"
	DefaultNodeSelectorBlockerLabelValue = ""
)

var (
	_ types.PodBlocker = &NodeSelectorPodBlocker{}
)

type NodeSelectorPodBlocker struct {
	nodeSelectorLabelName  string
	nodeSeelctorLabelValue string
}

func NewNodeSelectorPodBlocker() *NodeSelectorPodBlocker {
	return &NodeSelectorPodBlocker{
		nodeSelectorLabelName:  DefaultNodeSelectorBlockerLabelName,
		nodeSeelctorLabelValue: DefaultNodeSelectorBlockerLabelValue,
	}
}

func (b *NodeSelectorPodBlocker) Block(podSpec *corev1.PodSpec, logger logr.Logger) error {
	if podSpec.NodeSelector == nil {
		podSpec.NodeSelector = make(map[string]string)
	}
	podSpec.NodeSelector[b.nodeSelectorLabelName] = b.nodeSeelctorLabelValue

	return nil
}

func (b *NodeSelectorPodBlocker) Unblock(podSpec *corev1.PodSpec, logger logr.Logger) error {
	if podSpec.NodeSelector == nil {
		return nil
	}
	delete(podSpec.NodeSelector, b.nodeSelectorLabelName)

	return nil
}

func (b *NodeSelectorPodBlocker) IsBlocked(podSpec *corev1.PodSpec) bool {
	if podSpec.NodeSelector == nil {
		return false
	}
	_, ok := podSpec.NodeSelector[b.nodeSelectorLabelName]
	return ok
}
