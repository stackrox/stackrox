package k8sintrospect

import (
	"github.com/stackrox/rox/pkg/namespaces"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// DefaultConfig is the default configuration for the StackRox platform.
	DefaultConfig = Config{
		Namespaces: []string{namespaces.StackRox},
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
				GVK:           schema.GroupVersionKind{Version: "v1", Kind: "Secret"},
				RedactionFunc: RedactSecret,
				FilterFunc:    FilterOutServiceAccountSecrets,
			},
			{
				GVK: schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"},
			},
			{
				GVK: schema.GroupVersionKind{Version: "v1", Kind: "Service"},
			},
		},
	}
)
