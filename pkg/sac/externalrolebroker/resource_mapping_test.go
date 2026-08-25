package externalrolebroker

import (
	"testing"

	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
)

func TestAllResourcesAreMapped(t *testing.T) {
	mappedResources := set.NewSet[permissions.Resource]()
	for _, md := range resourceMapping {
		mappedResources.Add(md.GetResource())
	}

	for _, md := range resources.ListAllMetadata() {
		res := md.GetResource()
		assert.True(t, mappedResources.Contains(res),
			"Resource %q is defined in resources.ListAllMetadata() but has no entry in resourceMapping.", res)
	}
}
