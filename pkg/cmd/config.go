// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package cmd

import (
	"os"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type ExponentialPacer struct {
	MinInitial *int
	MaxStagger *int
	Multiplier *float64
}

type LinearPacer struct {
	MaxStagger *int
	Step       *int
}

type Pacer struct {
	Exponential *ExponentialPacer
	Linear      *LinearPacer
}

type StaggeringPolicy struct {
	Name                string
	LabelSelector       map[string]string
	BypassLabelSelector map[string]string
	GroupingExpression  string
	Pacer               Pacer
}

type Config struct {
	StaggeringPolicies []StaggeringPolicy
}

func LoadConfig(path string, logger logr.Logger) (Config, error) {
	logger.Info("loading config", "path", path)

	bytes, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	return LoadConfigFromString(string(bytes), logger)
}

func LoadConfigFromString(configString string, logger logr.Logger) (Config, error) {
	var config Config
	if err := yaml.Unmarshal([]byte(configString), &config); err != nil {
		return Config{}, err
	}

	return config, nil
}
