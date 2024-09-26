// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr/testr"
)

func TestLoadConfig_Success(t *testing.T) {
	logger := testr.New(t)

	// Create a temporary YAML config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := `
staggeringPolicies:
  - name: "policy1"
    labelSelector:
      app: "my-app"
    groupingExpression: "metadata.labels['app']"
    pacer:
      exponential:
        minInitial: 1
        maxStagger: 10
        multiplier: 1.5
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write temp config file: %v", err)
	}

	config, err := LoadConfig(configPath, logger)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(config.StaggeringPolicies) != 1 {
		t.Errorf("Expected 1 staggering policy, got %d", len(config.StaggeringPolicies))
	}

	policy := config.StaggeringPolicies[0]
	if policy.Name != "policy1" {
		t.Errorf("Expected policy name 'policy1', got '%s'", policy.Name)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	logger := testr.New(t)

	_, err := LoadConfig("nonexistent.yaml", logger)
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Expected file not found error, got %v", err)
	}
}

func TestLoadConfigFromString_Success(t *testing.T) {
	logger := testr.New(t)

	configString := `
staggeringPolicies:
  - name: "policy2"
    labelSelector:
      app: "another-app"
    groupingExpression: "metadata.labels['app']"
    pacer:
      exponential:
        minInitial: 2
        maxStagger: 20
        multiplier: 2.0
`

	config, err := LoadConfigFromString(configString, logger)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(config.StaggeringPolicies) != 1 {
		t.Errorf("Expected 1 staggering policy, got %d", len(config.StaggeringPolicies))
	}

	policy := config.StaggeringPolicies[0]
	if policy.Name != "policy2" {
		t.Errorf("Expected policy name 'policy2', got '%s'", policy.Name)
	}
}

func TestLoadConfigFromString_InvalidYAML(t *testing.T) {
	logger := testr.New(t)

	invalidConfigString := `
staggeringPolicies:
  - name: "policy3"
    labelSelector: [invalid_yaml
`

	_, err := LoadConfigFromString(invalidConfigString, logger)
	if err == nil {
		t.Error("Expected error due to invalid YAML, got nil")
	}
}

func TestLoadConfigFromString_EmptyString(t *testing.T) {
	logger := testr.New(t)

	_, err := LoadConfigFromString("", logger)
	if err != nil {
		t.Errorf("Expected no error for empty string, got %v", err)
	}
}
