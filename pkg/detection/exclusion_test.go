package detection

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchesDeploymentExclusion(t *testing.T) {
	cases := []struct {
		name        string
		deployment  *storage.Deployment
		policy      *storage.Policy
		shouldMatch bool
	}{
		{
			name:        "No excluded scope",
			deployment:  fixtures.GetDeployment(),
			policy:      &storage.Policy{},
			shouldMatch: false,
		},
		{
			name:       "Named excluded scope",
			deployment: fixtures.GetDeployment(),
			policy: &storage.Policy{
				Exclusions: []*storage.Exclusion{
					{Matcher: &storage.Exclusion_Deployment_{Deployment: &storage.Exclusion_Deployment{Name: fixtures.GetDeployment().GetName()}}},
				},
			},
			shouldMatch: true,
		},
		{
			name:       "Named excluded scope with matching regex",
			deployment: fixtures.GetDeployment(),
			policy: &storage.Policy{
				Exclusions: []*storage.Exclusion{
					{Matcher: &storage.Exclusion_Deployment_{Deployment: &storage.Exclusion_Deployment{Name: "nginx.*"}}},
				},
			},
			shouldMatch: true,
		},
		{
			name:       "Named excluded scope with non-matching regex",
			deployment: fixtures.GetDeployment(),
			policy: &storage.Policy{
				Exclusions: []*storage.Exclusion{
					{Matcher: &storage.Exclusion_Deployment_{Deployment: &storage.Exclusion_Deployment{Name: "nginy.*"}}},
				},
			},
			shouldMatch: false,
		},
		{
			name:       "Named excluded scope with invalid regex (ensure no error)",
			deployment: fixtures.GetDeployment(),
			policy: &storage.Policy{
				Exclusions: []*storage.Exclusion{
					{Matcher: &storage.Exclusion_Deployment_{Deployment: &storage.Exclusion_Deployment{Name: "ngin\\K"}}},
				},
			},
			shouldMatch: false,
		},
		{
			name:       "Named excluded scope, and another with a different name",
			deployment: fixtures.GetDeployment(),
			policy: &storage.Policy{
				Exclusions: []*storage.Exclusion{
					{Matcher: &storage.Exclusion_Deployment_{Deployment: &storage.Exclusion_Deployment{Name: fixtures.GetDeployment().GetName()}}},
					{Matcher: &storage.Exclusion_Deployment_{Deployment: &storage.Exclusion_Deployment{Name: uuid.NewV4().String()}}},
				},
			},
			shouldMatch: true,
		},
		{
			name:       "Named excluded scope with different name",
			deployment: fixtures.GetDeployment(),
			policy: &storage.Policy{
				Exclusions: []*storage.Exclusion{
					{Matcher: &storage.Exclusion_Deployment_{Deployment: &storage.Exclusion_Deployment{Name: uuid.NewV4().String()}}},
				},
			},
			shouldMatch: false,
		},
		{
			name:       "Scoped excluded scope",
			deployment: fixtures.GetDeployment(),
			policy: &storage.Policy{
				Exclusions: []*storage.Exclusion{
					{Matcher: &storage.Exclusion_Deployment_{Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: fixtures.GetDeployment().GetNamespace()}}}},
				},
			},
			shouldMatch: true,
		},
		{
			name:       "Scoped excluded scope with wrong name",
			deployment: fixtures.GetDeployment(),
			policy: &storage.Policy{
				Exclusions: []*storage.Exclusion{
					{Matcher: &storage.Exclusion_Deployment_{Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: uuid.NewV4().String()}}}},
				},
			},
			shouldMatch: false,
		},
		{
			name:       "Scoped excluded scope, but different name",
			deployment: fixtures.GetDeployment(),
			policy: &storage.Policy{
				Exclusions: []*storage.Exclusion{
					{Matcher: &storage.Exclusion_Deployment_{Deployment: &storage.Exclusion_Deployment{Name: uuid.NewV4().String(), Scope: &storage.Scope{Namespace: fixtures.GetDeployment().GetNamespace()}}}},
				},
			},
			shouldMatch: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			compiledExclusions := make([]*compiledExclusion, 0, len(c.policy.GetExclusions()))
			for _, w := range c.policy.GetExclusions() {
				cw, err := newCompiledExclusion(w)
				require.NoError(t, err)
				compiledExclusions = append(compiledExclusions, cw)
			}

			got := deploymentMatchesExclusions(context.Background(), c.deployment, compiledExclusions)
			assert.Equal(t, c.shouldMatch, got)
			// If it should match, make sure it doesn't match if the exclusions are all expired.
			if c.shouldMatch {
				for _, exclusion := range c.policy.GetExclusions() {
					exclusion.Expiration = protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour))
				}
				assert.False(t, deploymentMatchesExclusions(context.Background(), c.deployment, compiledExclusions))

				for _, exclusion := range c.policy.GetExclusions() {
					exclusion.Expiration = protoconv.MustConvertTimeToTimestamp(time.Now().Add(time.Hour))
				}
				assert.True(t, deploymentMatchesExclusions(context.Background(), c.deployment, compiledExclusions))
			}
			c.policy.Exclusions = append(c.policy.Exclusions, &storage.Exclusion{Image: &storage.Exclusion_Image{Name: "BLAH"}})
			assert.Equal(t, c.shouldMatch, got)
		})
	}
}

func TestMatchesWorkloadTypeExclusion(t *testing.T) {
	cronJob := fixtures.GetDeployment()
	cronJob.Type = kubernetes.CronJob
	job := fixtures.GetDeployment()
	job.Type = kubernetes.Job
	deployment := fixtures.GetDeployment()
	deployment.Type = kubernetes.Deployment

	typeExclusion := func(types ...storage.Exclusion_WorkloadType) *storage.Exclusion {
		return &storage.Exclusion{
			Matcher: &storage.Exclusion_ExcludeByType_{
				ExcludeByType: &storage.Exclusion_ExcludeByType{Types: types},
			},
		}
	}
	nameExclusion := func(name string) *storage.Exclusion {
		return &storage.Exclusion{
			Matcher: &storage.Exclusion_Deployment_{
				Deployment: &storage.Exclusion_Deployment{Name: name},
			},
		}
	}

	compile := func(t *testing.T, exclusions ...*storage.Exclusion) []*compiledExclusion {
		t.Helper()
		compiled := make([]*compiledExclusion, 0, len(exclusions))
		for _, exclusion := range exclusions {
			cw, err := newCompiledExclusion(exclusion)
			require.NoError(t, err)
			compiled = append(compiled, cw)
		}
		return compiled
	}

	t.Run("flag off does not exclude by type", func(t *testing.T) {
		testutils.MustUpdateFeature(t, features.PolicyWorkloadTypeExclusion, false)
		compiled := compile(t, typeExclusion(storage.Exclusion_CRON_JOB, storage.Exclusion_JOB))
		assert.False(t, deploymentMatchesExclusions(context.Background(), cronJob, compiled))
		assert.False(t, deploymentMatchesExclusions(context.Background(), job, compiled))
		assert.False(t, deploymentMatchesExclusions(context.Background(), deployment, compiled))
	})

	testutils.MustUpdateFeature(t, features.PolicyWorkloadTypeExclusion, true)

	cases := map[string]struct {
		exclusions  []*storage.Exclusion
		workload    *storage.Deployment
		shouldMatch bool
	}{
		"cronjob matches cronjob type": {
			exclusions:  []*storage.Exclusion{typeExclusion(storage.Exclusion_CRON_JOB)},
			workload:    cronJob,
			shouldMatch: true,
		},
		"deployment does not match cronjob type": {
			exclusions:  []*storage.Exclusion{typeExclusion(storage.Exclusion_CRON_JOB)},
			workload:    deployment,
			shouldMatch: false,
		},
		"job matches job and cronjob types": {
			exclusions:  []*storage.Exclusion{typeExclusion(storage.Exclusion_CRON_JOB, storage.Exclusion_JOB)},
			workload:    job,
			shouldMatch: true,
		},
		"OR with name/scope exclusion": {
			exclusions: []*storage.Exclusion{
				typeExclusion(storage.Exclusion_JOB),
				nameExclusion(deployment.GetName()),
			},
			workload:    deployment,
			shouldMatch: true,
		},
		"type exclusion does not match unrelated name": {
			exclusions:  []*storage.Exclusion{typeExclusion(storage.Exclusion_JOB)},
			workload:    deployment,
			shouldMatch: false,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			compiled := compile(t, c.exclusions...)
			got := deploymentMatchesExclusions(context.Background(), c.workload, compiled)
			assert.Equal(t, c.shouldMatch, got)
			if c.shouldMatch {
				for _, exclusion := range c.exclusions {
					exclusion.Expiration = protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour))
				}
				assert.False(t, deploymentMatchesExclusions(context.Background(), c.workload, compiled))
			}
		})
	}

	for name, tc := range map[string]struct {
		types       []storage.Exclusion_WorkloadType
		errContains string
	}{
		"empty type list is rejected": {
			errContains: "at least one workload type",
		},
		"unknown type is rejected": {
			types:       []storage.Exclusion_WorkloadType{storage.Exclusion_WorkloadType(99)},
			errContains: "unknown workload type",
		},
		"mixed valid and unknown types are rejected": {
			types:       []storage.Exclusion_WorkloadType{storage.Exclusion_JOB, storage.Exclusion_WorkloadType(99)},
			errContains: "unknown workload type",
		},
		"UNSPECIFIED is rejected": {
			types:       []storage.Exclusion_WorkloadType{storage.Exclusion_WORKLOAD_TYPE_UNSPECIFIED},
			errContains: "unknown workload type",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := newCompiledExclusion(typeExclusion(tc.types...))
			require.ErrorContains(t, err, tc.errContains)
		})
	}
}

func TestMatchesImageExclusion(t *testing.T) {
	cases := []struct {
		name        string
		image       string
		policy      *storage.Policy
		shouldMatch bool
	}{
		{
			name:  "no excluded scopes",
			image: "docker.io/stackrox/main",
			policy: &storage.Policy{
				Exclusions: []*storage.Exclusion{},
			},
			shouldMatch: false,
		},
		{
			name:  "doesn't match",
			image: "docker.io/stackrox/main",
			policy: &storage.Policy{
				Exclusions: []*storage.Exclusion{
					{Image: &storage.Exclusion_Image{Name: "docker.io/stackrox/mainasfasf"}},
				},
			},
			shouldMatch: false,
		},
		{
			name:  "matches",
			image: "docker.io/stackrox/main",
			policy: &storage.Policy{
				Exclusions: []*storage.Exclusion{
					{Image: &storage.Exclusion_Image{Name: "docker.io/stackrox/m"}},
				},
			},
			shouldMatch: true,
		},
		{
			name:  "one matches",
			image: "docker.io/stackrox/main",
			policy: &storage.Policy{
				Exclusions: []*storage.Exclusion{
					{Image: &storage.Exclusion_Image{Name: "BLAH"}},
					{Image: &storage.Exclusion_Image{Name: "docker.io/stackrox/m"}},
				},
			},
			shouldMatch: true,
		},
		{
			name:  "neither matches",
			image: "docker.io/stackrox/main",
			policy: &storage.Policy{
				Exclusions: []*storage.Exclusion{
					{Image: &storage.Exclusion_Image{Name: "BLAH"}},
					{Image: &storage.Exclusion_Image{Name: "docker.io/stackrox/masfasfa"}},
				},
			},
			shouldMatch: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := matchesImageExclusion(c.image, c.policy)
			assert.Equal(t, c.shouldMatch, got)
			// If it should match, make sure it doesn't match if the excluded scopes are all expired.
			if c.shouldMatch {
				for _, exclusion := range c.policy.GetExclusions() {
					exclusion.Expiration = protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour))
				}
				assert.False(t, matchesImageExclusion(c.image, c.policy))

				for _, exclusion := range c.policy.GetExclusions() {
					exclusion.Expiration = protoconv.MustConvertTimeToTimestamp(time.Now().Add(time.Hour))
				}
				assert.True(t, matchesImageExclusion(c.image, c.policy))
			}
			c.policy.Exclusions = append(c.policy.Exclusions, &storage.Exclusion{Matcher: &storage.Exclusion_Deployment_{Deployment: &storage.Exclusion_Deployment{Name: "BLAH"}}})
			assert.Equal(t, c.shouldMatch, got)
		})
	}
}
