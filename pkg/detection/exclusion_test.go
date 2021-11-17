package detection

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/protoconv"
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
					{
						Deployment: &storage.Exclusion_Deployment{Name: fixtures.GetDeployment().GetName()},
					},
				},
			},
			shouldMatch: true,
		},
		{
			name:       "Named excluded scope with matching regex",
			deployment: fixtures.GetDeployment(),
			policy: &storage.Policy{
				Exclusions: []*storage.Exclusion{
					{
						Deployment: &storage.Exclusion_Deployment{Name: "nginx.*"},
					},
				},
			},
			shouldMatch: true,
		},
		{
			name:       "Named excluded scope with non-matching regex",
			deployment: fixtures.GetDeployment(),
			policy: &storage.Policy{
				Exclusions: []*storage.Exclusion{
					{
						Deployment: &storage.Exclusion_Deployment{Name: "nginy.*"},
					},
				},
			},
			shouldMatch: false,
		},
		{
			name:       "Named excluded scope with invalid regex (ensure no error)",
			deployment: fixtures.GetDeployment(),
			policy: &storage.Policy{
				Exclusions: []*storage.Exclusion{
					{
						Deployment: &storage.Exclusion_Deployment{Name: "ngin\\K"},
					},
				},
			},
			shouldMatch: false,
		},
		{
			name:       "Named excluded scope, and another with a different name",
			deployment: fixtures.GetDeployment(),
			policy: &storage.Policy{
				Exclusions: []*storage.Exclusion{
					{
						Deployment: &storage.Exclusion_Deployment{Name: fixtures.GetDeployment().GetName()},
					},
					{
						Deployment: &storage.Exclusion_Deployment{Name: uuid.NewV4().String()},
					},
				},
			},
			shouldMatch: true,
		},
		{
			name:       "Named excluded scope with different name",
			deployment: fixtures.GetDeployment(),
			policy: &storage.Policy{
				Exclusions: []*storage.Exclusion{
					{
						Deployment: &storage.Exclusion_Deployment{Name: uuid.NewV4().String()},
					},
				},
			},
			shouldMatch: false,
		},
		{
			name:       "Scoped excluded scope",
			deployment: fixtures.GetDeployment(),
			policy: &storage.Policy{
				Exclusions: []*storage.Exclusion{
					{
						Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: fixtures.GetDeployment().GetNamespace()}},
					},
				},
			},
			shouldMatch: true,
		},
		{
			name:       "Scoped excluded scope with wrong name",
			deployment: fixtures.GetDeployment(),
			policy: &storage.Policy{
				Exclusions: []*storage.Exclusion{
					{
						Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: uuid.NewV4().String()}},
					},
				},
			},
			shouldMatch: false,
		},
		{
			name:       "Scoped excluded scope, but different name",
			deployment: fixtures.GetDeployment(),
			policy: &storage.Policy{
				Exclusions: []*storage.Exclusion{
					{
						Deployment: &storage.Exclusion_Deployment{Name: uuid.NewV4().String(), Scope: &storage.Scope{Namespace: fixtures.GetDeployment().GetNamespace()}},
					},
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

			got := deploymentMatchesExclusions(c.deployment, compiledExclusions)
			assert.Equal(t, c.shouldMatch, got)
			// If it should match, make sure it doesn't match if the exclusions are all expired.
			if c.shouldMatch {
				for _, exclusion := range c.policy.GetExclusions() {
					exclusion.Expiration = protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour))
				}
				assert.False(t, deploymentMatchesExclusions(c.deployment, compiledExclusions))

				for _, exclusion := range c.policy.GetExclusions() {
					exclusion.Expiration = protoconv.MustConvertTimeToTimestamp(time.Now().Add(time.Hour))
				}
				assert.True(t, deploymentMatchesExclusions(c.deployment, compiledExclusions))
			}
			c.policy.Exclusions = append(c.policy.Exclusions, &storage.Exclusion{Image: &storage.Exclusion_Image{Name: "BLAH"}})
			assert.Equal(t, c.shouldMatch, got)
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
			c.policy.Exclusions = append(c.policy.Exclusions, &storage.Exclusion{Deployment: &storage.Exclusion_Deployment{Name: "BLAH"}})
			assert.Equal(t, c.shouldMatch, got)
		})
	}
}
