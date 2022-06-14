package permissions

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestCheckResourceForAccess(t *testing.T) {
	testResource := Resource("Test")
	replacingResource := Resource("TestReplace")

	testResourceMetadata := ResourceMetadata{
		Resource: testResource,
	}
	testResourceWithReplacingResourceMetadata := ResourceMetadata{
		Resource: testResource,
		ReplacingResource: &ResourceMetadata{
			Resource: replacingResource,
		},
	}

	testResourceRead := map[string]storage.Access{string(testResource): storage.Access_READ_ACCESS}
	testResourceReadWrite := map[string]storage.Access{string(testResource): storage.Access_READ_WRITE_ACCESS}

	replacingResourceRead := map[string]storage.Access{string(replacingResource): storage.Access_READ_ACCESS}
	replacingResourceReadWrite := map[string]storage.Access{string(replacingResource): storage.Access_READ_WRITE_ACCESS}

	otherResourceRead := map[string]storage.Access{"Other": storage.Access_READ_ACCESS}

	cases := map[string]struct {
		resource    ResourceMetadata
		permissions map[string]storage.Access
		access      storage.Access
		expectedRes bool
	}{
		"non-replaced resource with READ access asking for READ access should return true": {
			resource:    testResourceMetadata,
			permissions: testResourceRead,
			access:      storage.Access_READ_ACCESS,
			expectedRes: true,
		},
		"non-replaced resource with WRITE access asking for READ access should return true": {
			resource:    testResourceMetadata,
			permissions: testResourceReadWrite,
			access:      storage.Access_READ_ACCESS,
			expectedRes: true,
		},
		"non-replaced resource with READ access asking for WRITE access should return false": {
			resource:    testResourceMetadata,
			permissions: testResourceRead,
			access:      storage.Access_READ_WRITE_ACCESS,
		},
		"non-replaced resource with no access asking for READ access should return false": {
			resource:    testResourceMetadata,
			permissions: otherResourceRead,
			access:      storage.Access_READ_ACCESS,
		},
		"replaced resource with READ access asking for READ access should return true": {
			resource:    testResourceWithReplacingResourceMetadata,
			permissions: replacingResourceRead,
			access:      storage.Access_READ_ACCESS,
			expectedRes: true,
		},
		"replaced resource with WRITE access asking for READ access should return true": {
			resource:    testResourceWithReplacingResourceMetadata,
			permissions: replacingResourceReadWrite,
			access:      storage.Access_READ_ACCESS,
			expectedRes: true,
		},
		"replaced resource with READ access asking for WRITE access should return false": {
			resource:    testResourceWithReplacingResourceMetadata,
			permissions: replacingResourceRead,
			access:      storage.Access_READ_WRITE_ACCESS,
		},
		"replaced resource with no access asking for READ access should return false": {
			resource:    testResourceWithReplacingResourceMetadata,
			permissions: otherResourceRead,
			access:      storage.Access_READ_ACCESS,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.expectedRes, c.resource.IsPermittedBy(c.permissions, c.access))
		})
	}
}
