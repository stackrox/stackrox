package main

import (
	"fmt"
	"os"
	"time"

	"go.yaml.in/yaml/v3"
)

type Config struct {
	Workload   WorkloadConfig   `yaml:"workload"`
	Policies   PoliciesConfig   `yaml:"policies"`
	Completion CompletionConfig `yaml:"completion"`
}

type WorkloadConfig struct {
	Namespaces   int               `yaml:"namespaces"`
	Deployments  DeploymentConfig  `yaml:"deployments"`
	Roles        RoleConfig        `yaml:"roles"`
	RoleBindings RoleBindingConfig `yaml:"roleBindings"`
	Services     ServiceConfig     `yaml:"services"`
}

type DeploymentConfig struct {
	Count          int    `yaml:"count"`
	Containers     int    `yaml:"containers"`
	Image          string `yaml:"image"`
	ServiceAccount string `yaml:"serviceAccount"`
}

type RoleConfig struct {
	Count       int      `yaml:"count"`
	ClusterWide bool     `yaml:"clusterWide"`
	Verbs       []string `yaml:"verbs"`
}

type RoleBindingConfig struct {
	Count int `yaml:"count"`
}

type ServiceConfig struct {
	Count int    `yaml:"count"`
	Type  string `yaml:"type"`
}

type PoliciesConfig struct {
	UseDefaults bool `yaml:"useDefaults"`
}

type CompletionConfig struct {
	Timeout    time.Duration     `yaml:"timeout"`
	Conditions []ConditionConfig `yaml:"conditions"`
}

type ConditionConfig struct {
	Type  string `yaml:"type"`
	Field string `yaml:"field"`
	Value string `yaml:"value"`
	Count int    `yaml:"count"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	applyDefaults(cfg)
	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}
	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Workload.Namespaces == 0 {
		cfg.Workload.Namespaces = 1
	}
	if cfg.Workload.Deployments.Containers == 0 {
		cfg.Workload.Deployments.Containers = 1
	}
	if cfg.Workload.Deployments.Image == "" {
		cfg.Workload.Deployments.Image = "nginx:1.25"
	}
	if cfg.Workload.Deployments.ServiceAccount == "" {
		cfg.Workload.Deployments.ServiceAccount = "default"
	}
	if len(cfg.Workload.Roles.Verbs) == 0 {
		cfg.Workload.Roles.Verbs = []string{"get", "list"}
	}
	if cfg.Workload.Services.Type == "" {
		cfg.Workload.Services.Type = "ClusterIP"
	}
	if cfg.Completion.Timeout == 0 {
		cfg.Completion.Timeout = 120 * time.Second
	}
}

func validate(cfg *Config) error {
	if cfg.Workload.Deployments.Count == 0 {
		return fmt.Errorf("workload.deployments.count must be > 0")
	}
	switch cfg.Workload.Services.Type {
	case "ClusterIP", "NodePort", "LoadBalancer":
	default:
		return fmt.Errorf("invalid service type %q: must be ClusterIP, NodePort, or LoadBalancer", cfg.Workload.Services.Type)
	}
	if len(cfg.Completion.Conditions) == 0 {
		return fmt.Errorf("at least one completion condition is required")
	}
	for i, c := range cfg.Completion.Conditions {
		switch c.Type {
		case "deploymentCount", "deploymentField":
		default:
			return fmt.Errorf("condition[%d]: unknown type %q", i, c.Type)
		}
		if c.Count <= 0 {
			return fmt.Errorf("condition[%d]: count must be > 0", i)
		}
		if c.Type == "deploymentField" && c.Field == "" {
			return fmt.Errorf("condition[%d]: deploymentField requires a field name", i)
		}
	}
	return nil
}
