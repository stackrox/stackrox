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
			policy: storage.Policy_builder{
				Exclusions: []*storage.Exclusion{
					storage.Exclusion_builder{
						Deployment: storage.Exclusion_Deployment_builder{Name: fixtures.GetDeployment().GetName()}.Build(),
					}.Build(),
				},
			}.Build(),
			shouldMatch: true,
		},
		{
			name:       "Named excluded scope with matching regex",
			deployment: fixtures.GetDeployment(),
			policy: storage.Policy_builder{
				Exclusions: []*storage.Exclusion{
					storage.Exclusion_builder{
						Deployment: storage.Exclusion_Deployment_builder{Name: "nginx.*"}.Build(),
					}.Build(),
				},
			}.Build(),
			shouldMatch: true,
		},
		{
			name:       "Named excluded scope with non-matching regex",
			deployment: fixtures.GetDeployment(),
			policy: storage.Policy_builder{
				Exclusions: []*storage.Exclusion{
					storage.Exclusion_builder{
						Deployment: storage.Exclusion_Deployment_builder{Name: "nginy.*"}.Build(),
					}.Build(),
				},
			}.Build(),
			shouldMatch: false,
		},
		{
			name:       "Named excluded scope with invalid regex (ensure no error)",
			deployment: fixtures.GetDeployment(),
			policy: storage.Policy_builder{
				Exclusions: []*storage.Exclusion{
					storage.Exclusion_builder{
						Deployment: storage.Exclusion_Deployment_builder{Name: "ngin\\K"}.Build(),
					}.Build(),
				},
			}.Build(),
			shouldMatch: false,
		},
		{
			name:       "Named excluded scope, and another with a different name",
			deployment: fixtures.GetDeployment(),
			policy: storage.Policy_builder{
				Exclusions: []*storage.Exclusion{
					storage.Exclusion_builder{
						Deployment: storage.Exclusion_Deployment_builder{Name: fixtures.GetDeployment().GetName()}.Build(),
					}.Build(),
					storage.Exclusion_builder{
						Deployment: storage.Exclusion_Deployment_builder{Name: uuid.NewV4().String()}.Build(),
					}.Build(),
				},
			}.Build(),
			shouldMatch: true,
		},
		{
			name:       "Named excluded scope with different name",
			deployment: fixtures.GetDeployment(),
			policy: storage.Policy_builder{
				Exclusions: []*storage.Exclusion{
					storage.Exclusion_builder{
						Deployment: storage.Exclusion_Deployment_builder{Name: uuid.NewV4().String()}.Build(),
					}.Build(),
				},
			}.Build(),
			shouldMatch: false,
		},
		{
			name:       "Scoped excluded scope",
			deployment: fixtures.GetDeployment(),
			policy: storage.Policy_builder{
				Exclusions: []*storage.Exclusion{
					storage.Exclusion_builder{
						Deployment: storage.Exclusion_Deployment_builder{Scope: storage.Scope_builder{Namespace: fixtures.GetDeployment().GetNamespace()}.Build()}.Build(),
					}.Build(),
				},
			}.Build(),
			shouldMatch: true,
		},
		{
			name:       "Scoped excluded scope with wrong name",
			deployment: fixtures.GetDeployment(),
			policy: storage.Policy_builder{
				Exclusions: []*storage.Exclusion{
					storage.Exclusion_builder{
						Deployment: storage.Exclusion_Deployment_builder{Scope: storage.Scope_builder{Namespace: uuid.NewV4().String()}.Build()}.Build(),
					}.Build(),
				},
			}.Build(),
			shouldMatch: false,
		},
		{
			name:       "Scoped excluded scope, but different name",
			deployment: fixtures.GetDeployment(),
			policy: storage.Policy_builder{
				Exclusions: []*storage.Exclusion{
					storage.Exclusion_builder{
						Deployment: storage.Exclusion_Deployment_builder{Name: uuid.NewV4().String(), Scope: storage.Scope_builder{Namespace: fixtures.GetDeployment().GetNamespace()}.Build()}.Build(),
					}.Build(),
				},
			}.Build(),
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
					exclusion.SetExpiration(protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)))
				}
				assert.False(t, deploymentMatchesExclusions(c.deployment, compiledExclusions))

				for _, exclusion := range c.policy.GetExclusions() {
					exclusion.SetExpiration(protoconv.MustConvertTimeToTimestamp(time.Now().Add(time.Hour)))
				}
				assert.True(t, deploymentMatchesExclusions(c.deployment, compiledExclusions))
			}
			ei := &storage.Exclusion_Image{}
			ei.SetName("BLAH")
			exclusion := &storage.Exclusion{}
			exclusion.SetImage(ei)
			c.policy.SetExclusions(append(c.policy.GetExclusions(), exclusion))
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
			policy: storage.Policy_builder{
				Exclusions: []*storage.Exclusion{},
			}.Build(),
			shouldMatch: false,
		},
		{
			name:  "doesn't match",
			image: "docker.io/stackrox/main",
			policy: storage.Policy_builder{
				Exclusions: []*storage.Exclusion{
					storage.Exclusion_builder{Image: storage.Exclusion_Image_builder{Name: "docker.io/stackrox/mainasfasf"}.Build()}.Build(),
				},
			}.Build(),
			shouldMatch: false,
		},
		{
			name:  "matches",
			image: "docker.io/stackrox/main",
			policy: storage.Policy_builder{
				Exclusions: []*storage.Exclusion{
					storage.Exclusion_builder{Image: storage.Exclusion_Image_builder{Name: "docker.io/stackrox/m"}.Build()}.Build(),
				},
			}.Build(),
			shouldMatch: true,
		},
		{
			name:  "one matches",
			image: "docker.io/stackrox/main",
			policy: storage.Policy_builder{
				Exclusions: []*storage.Exclusion{
					storage.Exclusion_builder{Image: storage.Exclusion_Image_builder{Name: "BLAH"}.Build()}.Build(),
					storage.Exclusion_builder{Image: storage.Exclusion_Image_builder{Name: "docker.io/stackrox/m"}.Build()}.Build(),
				},
			}.Build(),
			shouldMatch: true,
		},
		{
			name:  "neither matches",
			image: "docker.io/stackrox/main",
			policy: storage.Policy_builder{
				Exclusions: []*storage.Exclusion{
					storage.Exclusion_builder{Image: storage.Exclusion_Image_builder{Name: "BLAH"}.Build()}.Build(),
					storage.Exclusion_builder{Image: storage.Exclusion_Image_builder{Name: "docker.io/stackrox/masfasfa"}.Build()}.Build(),
				},
			}.Build(),
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
					exclusion.SetExpiration(protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)))
				}
				assert.False(t, matchesImageExclusion(c.image, c.policy))

				for _, exclusion := range c.policy.GetExclusions() {
					exclusion.SetExpiration(protoconv.MustConvertTimeToTimestamp(time.Now().Add(time.Hour)))
				}
				assert.True(t, matchesImageExclusion(c.image, c.policy))
			}
			ed := &storage.Exclusion_Deployment{}
			ed.SetName("BLAH")
			exclusion := &storage.Exclusion{}
			exclusion.SetDeployment(ed)
			c.policy.SetExclusions(append(c.policy.GetExclusions(), exclusion))
			assert.Equal(t, c.shouldMatch, got)
		})
	}
}
