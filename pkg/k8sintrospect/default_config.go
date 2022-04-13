package k8sintrospect

import (
	"github.com/stackrox/stackrox/pkg/env"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// DefaultConfig defines the default objects to pull in the diagnostic bundles
func DefaultConfig() Config {
	return Config{
		Namespaces: []string{env.Namespace.Setting()},
		Objects: []ObjectConfig{
			{
				GVK: schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
			},
			{
				GVK: schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "DaemonSet"},
			},
			{
				GVK: schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "ReplicaSet"},
			},
			{
				GVK: schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"},
			},
			{
				GVK: schema.GroupVersionKind{Version: "v1", Kind: "Service"},
			},
		},
	}
}

// DefaultConfigWithSecrets add secrets to the default objects to pull in the diagnostic bundles
func DefaultConfigWithSecrets() Config {
	cfg := DefaultConfig()
	cfg.Objects = append(cfg.Objects, ObjectConfig{
		GVK:           schema.GroupVersionKind{Version: "v1", Kind: "Secret"},
		RedactionFunc: RedactSecret,
		FilterFunc:    FilterOutServiceAccountSecrets,
	})
	return cfg
}
