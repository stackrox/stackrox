package matcher

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

// TestImageOnlyExclusionDoesNotExcludeEntities checks that an image-only exclusion leaves the policy
// applicable to every deployment, namespace, and cluster, while a deployment exclusion still excludes.
func TestImageOnlyExclusionDoesNotExcludeEntities(t *testing.T) {
	deployment := &storage.Deployment{Name: "payments", ClusterId: "cluster1", Namespace: "shop"}
	namespace := &storage.NamespaceMetadata{Name: "shop", ClusterId: "cluster1"}
	cluster := &storage.Cluster{Id: "cluster1"}

	matchers := map[string]Matcher{
		"deployment": NewDeploymentMatcher(deployment),
		"namespace":  NewNamespaceMatcher(namespace),
		"cluster":    NewClusterMatcher(cluster, []*storage.NamespaceMetadata{namespace}),
	}

	imageOnly := &storage.Policy{Exclusions: []*storage.Exclusion{
		{Image: &storage.Exclusion_Image{Name: "docker.io/library/nginx"}},
	}}
	clusterExclusion := &storage.Policy{Exclusions: []*storage.Exclusion{
		{Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Cluster: "cluster1"}}},
	}}

	for name, m := range matchers {
		t.Run(name, func(t *testing.T) {
			assert.True(t, m.IsPolicyApplicable(imageOnly), "image-only exclusion should not exclude")
			assert.False(t, m.IsPolicyApplicable(clusterExclusion), "cluster-scoped exclusion should exclude")
		})
	}
}
