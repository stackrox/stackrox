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

func TestMatchesDeploymentWhitelist(t *testing.T) {
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
				Whitelists: []*storage.Whitelist{
					{
						Deployment: &storage.Whitelist_Deployment{Name: fixtures.GetDeployment().GetName()},
					},
				},
			},
			shouldMatch: true,
		},
		{
			name:       "Named excluded scope, and another with a different name",
			deployment: fixtures.GetDeployment(),
			policy: &storage.Policy{
				Whitelists: []*storage.Whitelist{
					{
						Deployment: &storage.Whitelist_Deployment{Name: fixtures.GetDeployment().GetName()},
					},
					{
						Deployment: &storage.Whitelist_Deployment{Name: uuid.NewV4().String()},
					},
				},
			},
			shouldMatch: true,
		},
		{
			name:       "Named excluded scope with different name",
			deployment: fixtures.GetDeployment(),
			policy: &storage.Policy{
				Whitelists: []*storage.Whitelist{
					{
						Deployment: &storage.Whitelist_Deployment{Name: uuid.NewV4().String()},
					},
				},
			},
			shouldMatch: false,
		},
		{
			name:       "Scoped excluded scope",
			deployment: fixtures.GetDeployment(),
			policy: &storage.Policy{
				Whitelists: []*storage.Whitelist{
					{
						Deployment: &storage.Whitelist_Deployment{Scope: &storage.Scope{Namespace: fixtures.GetDeployment().GetNamespace()}},
					},
				},
			},
			shouldMatch: true,
		},
		{
			name:       "Scoped excluded scope with wrong name",
			deployment: fixtures.GetDeployment(),
			policy: &storage.Policy{
				Whitelists: []*storage.Whitelist{
					{
						Deployment: &storage.Whitelist_Deployment{Scope: &storage.Scope{Namespace: uuid.NewV4().String()}},
					},
				},
			},
			shouldMatch: false,
		},
		{
			name:       "Scoped excluded scope, but different name",
			deployment: fixtures.GetDeployment(),
			policy: &storage.Policy{
				Whitelists: []*storage.Whitelist{
					{
						Deployment: &storage.Whitelist_Deployment{Name: uuid.NewV4().String(), Scope: &storage.Scope{Namespace: fixtures.GetDeployment().GetNamespace()}},
					},
				},
			},
			shouldMatch: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			compiledWhitelists := make([]*compiledWhitelist, 0, len(c.policy.GetWhitelists()))
			for _, w := range c.policy.GetWhitelists() {
				cw, err := newCompiledWhitelist(w)
				require.NoError(t, err)
				compiledWhitelists = append(compiledWhitelists, cw)
			}

			got := deploymentMatchesWhitelists(c.deployment, compiledWhitelists)
			assert.Equal(t, c.shouldMatch, got)
			// If it should match, make sure it doesn't match if the whitelists are all expired.
			if c.shouldMatch {
				for _, whitelist := range c.policy.GetWhitelists() {
					whitelist.Expiration = protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour))
				}
				assert.False(t, deploymentMatchesWhitelists(c.deployment, compiledWhitelists))

				for _, whitelist := range c.policy.GetWhitelists() {
					whitelist.Expiration = protoconv.MustConvertTimeToTimestamp(time.Now().Add(time.Hour))
				}
				assert.True(t, deploymentMatchesWhitelists(c.deployment, compiledWhitelists))
			}
			c.policy.Whitelists = append(c.policy.Whitelists, &storage.Whitelist{Image: &storage.Whitelist_Image{Name: "BLAH"}})
			assert.Equal(t, c.shouldMatch, got)
		})
	}
}

func TestMatchesImageWhitelist(t *testing.T) {
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
				Whitelists: []*storage.Whitelist{},
			},
			shouldMatch: false,
		},
		{
			name:  "doesn't match",
			image: "docker.io/stackrox/main",
			policy: &storage.Policy{
				Whitelists: []*storage.Whitelist{
					{Image: &storage.Whitelist_Image{Name: "docker.io/stackrox/mainasfasf"}},
				},
			},
			shouldMatch: false,
		},
		{
			name:  "matches",
			image: "docker.io/stackrox/main",
			policy: &storage.Policy{
				Whitelists: []*storage.Whitelist{
					{Image: &storage.Whitelist_Image{Name: "docker.io/stackrox/m"}},
				},
			},
			shouldMatch: true,
		},
		{
			name:  "one matches",
			image: "docker.io/stackrox/main",
			policy: &storage.Policy{
				Whitelists: []*storage.Whitelist{
					{Image: &storage.Whitelist_Image{Name: "BLAH"}},
					{Image: &storage.Whitelist_Image{Name: "docker.io/stackrox/m"}},
				},
			},
			shouldMatch: true,
		},
		{
			name:  "neither matches",
			image: "docker.io/stackrox/main",
			policy: &storage.Policy{
				Whitelists: []*storage.Whitelist{
					{Image: &storage.Whitelist_Image{Name: "BLAH"}},
					{Image: &storage.Whitelist_Image{Name: "docker.io/stackrox/masfasfa"}},
				},
			},
			shouldMatch: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := matchesImageWhitelist(c.image, c.policy)
			assert.Equal(t, c.shouldMatch, got)
			// If it should match, make sure it doesn't match if the excluded scopes are all expired.
			if c.shouldMatch {
				for _, whitelist := range c.policy.GetWhitelists() {
					whitelist.Expiration = protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour))
				}
				assert.False(t, matchesImageWhitelist(c.image, c.policy))

				for _, whitelist := range c.policy.GetWhitelists() {
					whitelist.Expiration = protoconv.MustConvertTimeToTimestamp(time.Now().Add(time.Hour))
				}
				assert.True(t, matchesImageWhitelist(c.image, c.policy))
			}
			c.policy.Whitelists = append(c.policy.Whitelists, &storage.Whitelist{Deployment: &storage.Whitelist_Deployment{Name: "BLAH"}})
			assert.Equal(t, c.shouldMatch, got)
		})
	}
}
