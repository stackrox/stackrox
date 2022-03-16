package main

import (
	"fmt"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
)

var typeRegistry = make(map[string]string)

func init() {
	for s, r := range map[proto.Message]permissions.ResourceHandle{
		&storage.ClusterHealthStatus{}:    resources.Cluster,
		&storage.ImageComponentEdge{}:     resources.ImageComponent,
		&storage.K8SRoleBinding{}:         resources.K8sRoleBinding,
		&storage.K8SRole{}:                resources.K8sRole,
		&storage.NamespaceMetadata{}:      resources.Namespace,
		&storage.NetworkEntity{}:          resources.NetworkGraph,
		&storage.PermissionSet{}:          resources.Role,
		&storage.Pod{}:                    resources.Deployment,
		&storage.ProcessBaselineResults{}: resources.ProcessWhitelist,
		&storage.ProcessBaseline{}:        resources.ProcessWhitelist,
		&storage.ProcessIndicator{}:       resources.Indicator,
		&storage.ReportConfiguration{}:    resources.VulnerabilityReports,
		&storage.SimpleAccessScope{}:      resources.Role,
		&storage.TokenMetadata{}:          resources.APIToken,
		// Tests
		&storage.TestMultiKeyStruct{}:  resources.Namespace,
		&storage.TestSingleKeyStruct{}: resources.Namespace,
	} {
		typeRegistry[fmt.Sprintf("%T", s)] = string(r.GetResource())
	}
}

func storageToResource(t string) string {
	if !strings.HasPrefix(t, "*") {
		t = "*" + t
	}
	s, ok := typeRegistry[t]
	if ok {
		return s
	}
	return strings.TrimPrefix(t, "*storage.")
}

func isGloballyScoped(storageType string) bool {
	resource := storageToResource(storageType)
	for _, resourceMetadata := range resources.ListAllMetadata() {
		if string(resourceMetadata.Resource) == resource {
			return resourceMetadata.Scope == permissions.GlobalScope
		}
	}
	return false
}
