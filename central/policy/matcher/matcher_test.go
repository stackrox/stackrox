package matcher

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

// TestImageOnlyExclusionDoesNotExcludeEntities checks that an image-only exclusion leaves the policy
// applicable to every deployment, namespace, and cluster, while a deployment exclusion still excludes.
func TestImageOnlyExclusionDoesNotExcludeEntities(t *testing.T) {
	ctx := context.Background()
	deployment := &storage.Deployment{Name: "payments", ClusterId: "cluster1", Namespace: "shop"}
	namespace := &storage.NamespaceMetadata{Name: "shop", ClusterId: "cluster1"}
	cluster := &storage.Cluster{Id: "cluster1"}

	matchers := map[string]Matcher{
		"deployment": NewDeploymentMatcher(deployment, nil, nil),
		"namespace":  NewNamespaceMatcher(namespace),
		"cluster":    NewClusterMatcher(cluster, []*storage.NamespaceMetadata{namespace}),
	}

	imageOnly := &storage.Policy{Exclusions: []*storage.Exclusion{
		{Image: &storage.Exclusion_Image{Name: "docker.io/library/nginx"}},
	}}
	clusterExclusion := &storage.Policy{Exclusions: []*storage.Exclusion{
		{Matcher: &storage.Exclusion_Deployment_{Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Cluster: "cluster1"}}}},
	}}

	for name, m := range matchers {
		t.Run(name, func(t *testing.T) {
			assert.True(t, m.IsPolicyApplicable(ctx, imageOnly), "image-only exclusion should not exclude")
			assert.False(t, m.IsPolicyApplicable(ctx, clusterExclusion), "cluster-scoped exclusion should exclude")
		})
	}
}

func TestKindExclusionExcludesMatchingDeploymentsOnly(t *testing.T) {
	testutils.MustUpdateFeature(t, features.PolicyWorkloadKindExclusion, true)
	ctx := context.Background()
	namespace := &storage.NamespaceMetadata{Name: "shop", ClusterId: "cluster1"}
	cluster := &storage.Cluster{Id: "cluster1"}
	policy := &storage.Policy{Exclusions: []*storage.Exclusion{{
		Matcher: &storage.Exclusion_ExcludeByKind_{
			ExcludeByKind: &storage.Exclusion_ExcludeByKind{
				Kinds: []storage.Exclusion_WorkloadKind{storage.Exclusion_JOB, storage.Exclusion_CRON_JOB},
			},
		},
	}}}

	job := &storage.Deployment{Name: "batch", Type: kubernetes.Job, ClusterId: "cluster1", Namespace: "shop"}
	cronJob := &storage.Deployment{Name: "nightly", Type: kubernetes.CronJob, ClusterId: "cluster1", Namespace: "shop"}
	deploy := &storage.Deployment{Name: "web", Type: kubernetes.Deployment, ClusterId: "cluster1", Namespace: "shop"}

	assert.False(t, NewDeploymentMatcher(job, nil, nil).IsPolicyApplicable(ctx, policy))
	assert.False(t, NewDeploymentMatcher(cronJob, nil, nil).IsPolicyApplicable(ctx, policy))
	assert.True(t, NewDeploymentMatcher(deploy, nil, nil).IsPolicyApplicable(ctx, policy))
	assert.True(t, NewNamespaceMatcher(namespace).IsPolicyApplicable(ctx, policy))
	assert.True(t, NewClusterMatcher(cluster, []*storage.NamespaceMetadata{namespace}).IsPolicyApplicable(ctx, policy))
}

func TestKindExclusionFlagOffLeavesPolicyApplicable(t *testing.T) {
	testutils.MustUpdateFeature(t, features.PolicyWorkloadKindExclusion, false)
	job := &storage.Deployment{Name: "batch", Type: kubernetes.Job, ClusterId: "cluster1", Namespace: "shop"}
	policy := &storage.Policy{Exclusions: []*storage.Exclusion{{
		Matcher: &storage.Exclusion_ExcludeByKind_{
			ExcludeByKind: &storage.Exclusion_ExcludeByKind{
				Kinds: []storage.Exclusion_WorkloadKind{storage.Exclusion_JOB},
			},
		},
	}}}

	assert.True(t, NewDeploymentMatcher(job, nil, nil).IsPolicyApplicable(context.Background(), policy))
}
