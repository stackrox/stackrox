package compiledpolicies

import (
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	containerPredicate "github.com/stackrox/rox/pkg/compiledpolicies/container/predicate"
	"github.com/stretchr/testify/assert"
)

func TestLatestImageTagPolicy(t *testing.T) {
	t.Parallel()

	policy := &v1.Policy{
		Name:       "latest",
		Severity:   v1.Severity_LOW_SEVERITY,
		Categories: []string{"Image Assurance"},
		Fields: &v1.PolicyFields{
			ImageName: &v1.ImageNamePolicy{
				Tag: "latest",
			},
		},
	}
	deployment := &v1.Deployment{
		Containers: []*v1.Container{
			{
				Image: &v1.Image{
					Name: &v1.ImageName{
						Tag:    "latest",
						Remote: "stackrox/health",
					},
				},
				SecurityContext: &v1.SecurityContext{
					Privileged: false,
				},
			},
		},
	}
	numViolations := 1

	p, err := New(policy)

	assert.NoError(t, err)
	assert.NotNil(t, p)

	actualViolations := p.Match(deployment)
	actualExcluded := p.Excluded(deployment)

	assert.Equal(t, numViolations, len(actualViolations))
	assert.Equal(t, (*v1.DryRunResponse_Excluded)(nil), actualExcluded)
}

func TestLatestImageTagPolicyNotPrivileged(t *testing.T) {
	t.Parallel()

	policy := &v1.Policy{
		Name:       "latest",
		Severity:   v1.Severity_LOW_SEVERITY,
		Categories: []string{"Image Assurance", "Privileges Capabilities"},
		Fields: &v1.PolicyFields{
			ImageName: &v1.ImageNamePolicy{
				Tag: "latest",
			},
			SetPrivileged: &v1.PolicyFields_Privileged{
				Privileged: true,
			},
		},
	}
	deployment := &v1.Deployment{
		Containers: []*v1.Container{
			{
				Image: &v1.Image{
					Name: &v1.ImageName{
						Tag:    "latest",
						Remote: "stackrox/health",
					},
				},
				SecurityContext: &v1.SecurityContext{
					Privileged: false,
				},
			},
		},
	}
	numViolations := 0

	p, err := New(policy)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	actualViolations := p.Match(deployment)
	actualExcluded := p.Excluded(deployment)

	assert.Equal(t, numViolations, len(actualViolations))
	assert.Equal(t, (*v1.DryRunResponse_Excluded)(nil), actualExcluded)
}

func TestLatestImageTagPolicyLatestNotPrivileged(t *testing.T) {
	t.Parallel()

	policy := &v1.Policy{
		Name:       "latest",
		Severity:   v1.Severity_LOW_SEVERITY,
		Categories: []string{"Image Assurance", "Privileges Capabilities"},
		Fields: &v1.PolicyFields{
			ImageName: &v1.ImageNamePolicy{
				Tag: "latest",
			},
			SetPrivileged: &v1.PolicyFields_Privileged{
				Privileged: true,
			},
		},
	}
	deployment := &v1.Deployment{
		Containers: []*v1.Container{
			{
				Image: &v1.Image{
					Name: &v1.ImageName{
						Tag:    "latest",
						Remote: "stackrox/health",
					},
				},
				SecurityContext: &v1.SecurityContext{
					Privileged: false,
				},
			},
			{
				Image: &v1.Image{
					Name: &v1.ImageName{
						Tag:    "1.4",
						Remote: "stackrox/health",
					},
				},
				SecurityContext: &v1.SecurityContext{
					Privileged: true,
				},
			},
		},
	}
	numViolations := 0

	p, err := New(policy)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	actualViolations := p.Match(deployment)
	actualExcluded := p.Excluded(deployment)

	assert.Equal(t, numViolations, len(actualViolations))
	assert.Equal(t, (*v1.DryRunResponse_Excluded)(nil), actualExcluded)
}

func TestLatestImageTagPolicyPrivilegedAndMatch(t *testing.T) {
	t.Parallel()

	policy := &v1.Policy{
		Name:       "latest",
		Severity:   v1.Severity_LOW_SEVERITY,
		Categories: []string{"Image Assurance", "Privileges Capabilities"},
		Fields: &v1.PolicyFields{
			ImageName: &v1.ImageNamePolicy{
				Tag: "latest",
			},
			SetPrivileged: &v1.PolicyFields_Privileged{
				Privileged: true,
			},
		},
	}
	deployment := &v1.Deployment{
		Containers: []*v1.Container{
			{
				Image: &v1.Image{
					Name: &v1.ImageName{
						Tag:    "latest",
						Remote: "stackrox/health",
					},
				},
				SecurityContext: &v1.SecurityContext{
					Privileged: true,
				},
			},
			{
				Image: &v1.Image{
					Name: &v1.ImageName{
						Tag:    "1.5",
						Remote: "stackrox/zookeeper",
					},
				},
				SecurityContext: &v1.SecurityContext{
					Privileged: true,
				},
			},
		},
	}
	numViolations := 2

	p, err := New(policy)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	actualViolations := p.Match(deployment)
	actualExcluded := p.Excluded(deployment)

	assert.Equal(t, numViolations, len(actualViolations))
	assert.Equal(t, (*v1.DryRunResponse_Excluded)(nil), actualExcluded)
}

func TestLatestImageTagPolicyAndTwoMatches(t *testing.T) {
	t.Parallel()

	policy := &v1.Policy{
		Name:       "latest",
		Severity:   v1.Severity_LOW_SEVERITY,
		Categories: []string{"Image Assurance", "Privileges Capabilities"},
		Fields: &v1.PolicyFields{
			ImageName: &v1.ImageNamePolicy{
				Tag: "latest",
			},
			SetPrivileged: &v1.PolicyFields_Privileged{
				Privileged: true,
			},
		},
	}
	deployment := &v1.Deployment{
		Containers: []*v1.Container{
			{
				Image: &v1.Image{
					Name: &v1.ImageName{
						Tag:    "latest",
						Remote: "stackrox/health",
					},
				},
				SecurityContext: &v1.SecurityContext{
					Privileged: true,
				},
			},
			{
				Image: &v1.Image{
					Name: &v1.ImageName{
						Tag:    "latest",
						Remote: "stackrox/zookeeper",
					},
				},
				SecurityContext: &v1.SecurityContext{
					Privileged: true,
				},
			},
		},
	}
	numViolations := 4

	p, err := New(policy)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	actualViolations := p.Match(deployment)
	actualExcluded := p.Excluded(deployment)

	assert.Equal(t, numViolations, len(actualViolations))
	assert.Equal(t, (*v1.DryRunResponse_Excluded)(nil), actualExcluded)
}

func TestLatestImageTagPolicyUnmatchedWhitelists(t *testing.T) {
	t.Parallel()

	policy := &v1.Policy{
		Name:       "latest",
		Severity:   v1.Severity_LOW_SEVERITY,
		Categories: []string{"Image Assurance"},
		Fields: &v1.PolicyFields{
			ImageName: &v1.ImageNamePolicy{
				Tag: "latest",
			},
		},
		Whitelists: []*v1.Whitelist{
			{
				Container: &v1.Whitelist_Container{
					ImageName: &v1.ImageName{
						Remote: "stackrox/kafka",
					},
				},
			},
			{
				Deployment: &v1.Whitelist_Deployment{
					Scope: &v1.Scope{
						Namespace: "blah",
					},
				},
			},
		},
	}
	deployment := &v1.Deployment{
		Containers: []*v1.Container{
			{
				Image: &v1.Image{
					Name: &v1.ImageName{
						Tag:    "latest",
						Remote: "stackrox/health",
					},
				},
				SecurityContext: &v1.SecurityContext{
					Privileged: false,
				},
			},
		},
	}
	numViolations := 1

	p, err := New(policy)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	actualViolations := p.Match(deployment)
	actualExcluded := p.Excluded(deployment)

	assert.Equal(t, numViolations, len(actualViolations))
	assert.Equal(t, (*v1.DryRunResponse_Excluded)(nil), actualExcluded)
}

func TestLatestImageTagPolicyMatchedWhitelist(t *testing.T) {
	t.Parallel()

	policy := &v1.Policy{
		Name:       "latest",
		Severity:   v1.Severity_LOW_SEVERITY,
		Categories: []string{"Image Assurance"},
		Fields: &v1.PolicyFields{
			ImageName: &v1.ImageNamePolicy{
				Tag: "latest",
			},
		},
		Whitelists: []*v1.Whitelist{
			{
				Container: &v1.Whitelist_Container{
					ImageName: &v1.ImageName{
						Remote: "stackrox/kafka",
					},
				},
			},
			{
				Deployment: &v1.Whitelist_Deployment{
					Name: "deployment1",
				},
			},
		},
	}
	deployment := &v1.Deployment{
		Name: "deployment1",
		Containers: []*v1.Container{
			{
				Image: &v1.Image{
					Name: &v1.ImageName{
						Tag:    "latest",
						Remote: "stackrox/health",
					},
				},
			},
		},
	}
	excluded := &v1.DryRunResponse_Excluded{
		Deployment: "deployment1",
		Whitelist: &v1.Whitelist{
			Deployment: &v1.Whitelist_Deployment{
				Name: "deployment1",
			},
		},
	}
	numViolations := 0

	p, err := New(policy)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	actualViolations := p.Match(deployment)
	actualExcluded := p.Excluded(deployment)

	assert.Equal(t, numViolations, len(actualViolations))
	assert.Equal(t, excluded, actualExcluded)
}

func TestLatestImageTagPolicyContainerMatchesWhitelist(t *testing.T) {
	t.Parallel()

	policy := &v1.Policy{
		Name:       "latest",
		Severity:   v1.Severity_LOW_SEVERITY,
		Categories: []string{"Image Assurance"},
		Fields: &v1.PolicyFields{
			ImageName: &v1.ImageNamePolicy{
				Tag: "latest",
			},
		},
		Whitelists: []*v1.Whitelist{
			{
				Container: &v1.Whitelist_Container{
					ImageName: &v1.ImageName{
						Remote: "stackrox/kafka",
					},
				},
			},
			{
				Container: &v1.Whitelist_Container{
					ImageName: &v1.ImageName{
						Tag:    "latest",
						Remote: "stackrox/health",
					},
				},
			},
		},
	}
	deployment := &v1.Deployment{
		Name: "deployment1",
		Containers: []*v1.Container{
			{
				Image: &v1.Image{
					Name: &v1.ImageName{
						Tag:    "latest",
						Remote: "stackrox/health",
					},
				},
			},
		},
	}
	numViolations := 0

	p, err := New(policy)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	actualViolations := p.Match(deployment)
	actualExcluded := p.Excluded(deployment)

	assert.Equal(t, numViolations, len(actualViolations))
	assert.Equal(t, (*v1.DryRunResponse_Excluded)(nil), actualExcluded)
}

func TestScopeDisabled(t *testing.T) {
	t.Parallel()

	policy := &v1.Policy{
		Disabled: true,
	}
	deployment := &v1.Deployment{
		ClusterId: "clusterA",
		Namespace: "namespace",
		Labels: map[string]string{
			"key": "value",
			"foo": "bar",
		},
		Containers: []*v1.Container{
			{
				Image: &v1.Image{
					Name: &v1.ImageName{
						Tag:    "latest",
						Remote: "stackrox/health",
					},
				},
			},
		},
	}

	p, err := New(policy)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	assert.False(t, p.ShouldProcess(deployment))
}

func TestScopeWrongCluster(t *testing.T) {
	t.Parallel()

	policy := &v1.Policy{
		Scope: []*v1.Scope{
			{
				Cluster: "clusterB",
			},
		},
	}
	deployment := &v1.Deployment{
		ClusterId: "clusterA",
		Namespace: "namespace",
		Labels: map[string]string{
			"key": "value",
			"foo": "bar",
		},
		Containers: []*v1.Container{
			{
				Image: &v1.Image{
					Name: &v1.ImageName{
						Tag:    "latest",
						Remote: "stackrox/health",
					},
				},
			},
		},
	}

	p, err := New(policy)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	assert.False(t, p.ShouldProcess(deployment))
}

func TestScopeWrongNamespace(t *testing.T) {
	t.Parallel()

	policy := &v1.Policy{
		Scope: []*v1.Scope{
			{
				Cluster:   "clusterA",
				Namespace: "notanamespace",
			},
		},
	}
	deployment := &v1.Deployment{
		ClusterId: "clusterA",
		Namespace: "namespace",
		Labels: map[string]string{
			"key": "value",
			"foo": "bar",
		},
		Containers: []*v1.Container{
			{
				Image: &v1.Image{
					Name: &v1.ImageName{
						Tag:    "latest",
						Remote: "stackrox/health",
					},
				},
			},
		},
	}

	p, err := New(policy)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	assert.False(t, p.ShouldProcess(deployment))
}

func TestScopeWrongLabel(t *testing.T) {
	t.Parallel()

	policy := &v1.Policy{
		Scope: []*v1.Scope{
			{
				Cluster:   "clusterA",
				Namespace: "namespace",
				Label: &v1.Scope_Label{
					Key:   "foo",
					Value: "car",
				},
			},
		},
	}
	deployment := &v1.Deployment{
		ClusterId: "clusterA",
		Namespace: "namespace",
		Labels: map[string]string{
			"key": "value",
			"foo": "bar",
		},
		Containers: []*v1.Container{
			{
				Image: &v1.Image{
					Name: &v1.ImageName{
						Tag:    "latest",
						Remote: "stackrox/health",
					},
				},
			},
		},
	}

	p, err := New(policy)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	// Predicate should fail since deployment is not in any scope.
	assert.False(t, p.ShouldProcess(deployment))
}

func TestScopeMatchesOnlyNamespace(t *testing.T) {
	t.Parallel()

	policy := &v1.Policy{
		Scope: []*v1.Scope{
			{
				Namespace: "namespace",
			},
		},
	}
	deployment := &v1.Deployment{
		ClusterId: "clusterA",
		Namespace: "namespace",
		Labels: map[string]string{
			"key": "value",
			"foo": "bar",
		},
		Containers: []*v1.Container{
			{
				Image: &v1.Image{
					Name: &v1.ImageName{
						Tag:    "latest",
						Remote: "stackrox/health",
					},
				},
			},
		},
	}

	p, err := New(policy)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	// Predicate should fail since deployment matches scope.
	assert.True(t, p.ShouldProcess(deployment))
}

func TestScopeMatchesAll(t *testing.T) {
	t.Parallel()

	policy := &v1.Policy{
		Scope: []*v1.Scope{
			{
				Cluster:   "clusterA",
				Namespace: "namespace",
				Label: &v1.Scope_Label{
					Key:   "foo",
					Value: "bar",
				},
			},
		},
	}
	deployment := &v1.Deployment{
		ClusterId: "clusterA",
		Namespace: "namespace",
		Labels: map[string]string{
			"key": "value",
			"foo": "bar",
		},
		Containers: []*v1.Container{
			{
				Image: &v1.Image{
					Name: &v1.ImageName{
						Tag:    "latest",
						Remote: "stackrox/health",
					},
				},
			},
		},
	}

	p, err := New(policy)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	// Predicate should pass since deployment matches scope.
	assert.True(t, p.ShouldProcess(deployment))
}

func TestScopeMatchesOneScope(t *testing.T) {
	t.Parallel()

	policy := &v1.Policy{
		Scope: []*v1.Scope{
			{
				Cluster: "clusterA",
			},
			{
				Cluster:   "clusterB",
				Namespace: "namespace",
			},
		},
	}
	deployment := &v1.Deployment{
		ClusterId: "clusterA",
		Namespace: "namespace",
		Labels: map[string]string{
			"key": "value",
			"foo": "bar",
		},
		Containers: []*v1.Container{
			{
				Image: &v1.Image{
					Name: &v1.ImageName{
						Tag:    "latest",
						Remote: "stackrox/health",
					},
				},
			},
		},
	}

	p, err := New(policy)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	// Predicate should pass since deployment matches a scope.
	assert.True(t, p.ShouldProcess(deployment))
}

func TestWhitelistMatchesScope(t *testing.T) {
	t.Parallel()

	policy := &v1.Policy{
		Whitelists: []*v1.Whitelist{
			{
				Deployment: &v1.Whitelist_Deployment{
					Scope: &v1.Scope{
						Cluster:   "clusterA",
						Namespace: "namespace",
						Label: &v1.Scope_Label{
							Key:   "foo",
							Value: "bar",
						},
					},
				},
			},
		},
	}
	deployment := &v1.Deployment{
		ClusterId: "clusterA",
		Namespace: "namespace",
		Labels: map[string]string{
			"key": "value",
			"foo": "bar",
		},
		Containers: []*v1.Container{
			{
				Image: &v1.Image{
					Name: &v1.ImageName{
						Tag:    "latest",
						Remote: "stackrox/health",
					},
				},
			},
		},
	}

	p, err := New(policy)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	// Predicate should fail since deployment matches whitelist.
	assert.False(t, p.ShouldProcess(deployment))
}

func TestWhitelistMatchesServiceName(t *testing.T) {
	t.Parallel()

	policy := &v1.Policy{
		Whitelists: []*v1.Whitelist{
			{
				Deployment: &v1.Whitelist_Deployment{
					Name: "deployment1",
				},
			},
		},
	}
	deployment := &v1.Deployment{
		Name: "deployment1",
	}

	p, err := New(policy)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	// Predicate should fail since deployment matches whitelist.
	assert.False(t, p.ShouldProcess(deployment))
}

func TestContainerWhitelistMatchesRegistry(t *testing.T) {
	t.Parallel()

	policy := &v1.Policy{
		Whitelists: []*v1.Whitelist{
			{
				Container: &v1.Whitelist_Container{
					ImageName: &v1.ImageName{
						Registry: "registry",
					},
				},
			},
		},
	}
	container := &v1.Container{
		Image: &v1.Image{
			Name: &v1.ImageName{
				Registry: "registry",
			},
		},
	}

	p, err := containerPredicate.Compile(policy)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	// Predicate should fail since container matches whitelist.
	assert.False(t, p(container))
}

func TestContainerWhitelistMatchesOneButNotOthers(t *testing.T) {
	t.Parallel()

	policy := &v1.Policy{
		Whitelists: []*v1.Whitelist{
			{
				Container: &v1.Whitelist_Container{
					ImageName: &v1.ImageName{
						Registry: "registry",
						Remote:   "remote",
					},
				},
			},
		},
	}
	container := &v1.Container{
		Image: &v1.Image{
			Name: &v1.ImageName{
				Registry: "registry1",
				Remote:   "remote",
			},
		},
	}

	p, err := containerPredicate.Compile(policy)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	// Predicate should pass since container does not match the whitelist.
	assert.True(t, p(container))
}

func TestContainerWhitelistMatchesAll(t *testing.T) {
	t.Parallel()

	policy := &v1.Policy{
		Whitelists: []*v1.Whitelist{
			{
				Container: &v1.Whitelist_Container{
					ImageName: &v1.ImageName{
						Sha:      "sha",
						Registry: "registry",
						Remote:   "remote",
						Tag:      "tag",
					},
				},
			},
		},
	}
	container := &v1.Container{
		Image: &v1.Image{
			Name: &v1.ImageName{
				Sha:      "sha",
				Registry: "registry",
				Remote:   "remote",
				Tag:      "tag",
			},
		},
	}

	p, err := containerPredicate.Compile(policy)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	// Predicate should fail since container matches whitelist.
	assert.False(t, p(container))
}
