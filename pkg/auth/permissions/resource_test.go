package permissions

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestCheckResourceForAccess(t *testing.T) {
	cases := map[string]struct {
		resource    ResourceMetadata
		permissions map[string]storage.Access
		access      storage.Access
		expectedRes bool
	}{
		"non-replaced resource with READ access asking for READ access should return true": {
			resource: ResourceMetadata{
				Resource: "Test",
			},
			permissions: map[string]storage.Access{"Test": storage.Access_READ_ACCESS},
			access:      storage.Access_READ_ACCESS,
			expectedRes: true,
		},
		"non-replaced resource with WRITE access asking for READ access should return true": {
			resource: ResourceMetadata{
				Resource: "Test",
			},
			permissions: map[string]storage.Access{"Test": storage.Access_READ_WRITE_ACCESS},
			access:      storage.Access_READ_ACCESS,
			expectedRes: true,
		},
		"non-replaced resource with READ access asking for WRITE access should return false": {
			resource: ResourceMetadata{
				Resource: "Test",
			},
			permissions: map[string]storage.Access{"Test": storage.Access_READ_ACCESS},
			access:      storage.Access_READ_WRITE_ACCESS,
		},
		"non-replaced resource with no access asking for READ access should return false": {
			resource: ResourceMetadata{
				Resource: "Test",
			},
			permissions: map[string]storage.Access{"Other resource": storage.Access_READ_ACCESS},
			access:      storage.Access_READ_WRITE_ACCESS,
		},
		"replaced resource with READ access asking for READ access should return true": {
			resource: ResourceMetadata{
				Resource: "Test",
				ReplacingResource: &ResourceMetadata{
					Resource: "ReplaceTest",
				},
			},
			permissions: map[string]storage.Access{"ReplaceTest": storage.Access_READ_ACCESS},
			access:      storage.Access_READ_ACCESS,
			expectedRes: true,
		},
		"replaced resource with WRITE access asking for READ access should return true": {
			resource: ResourceMetadata{
				Resource: "Test",
				ReplacingResource: &ResourceMetadata{
					Resource: "ReplaceTest",
				},
			},
			permissions: map[string]storage.Access{"ReplaceTest": storage.Access_READ_WRITE_ACCESS},
			access:      storage.Access_READ_ACCESS,
			expectedRes: true,
		},
		"replaced resource with READ access asking for WRITE access should return false": {
			resource: ResourceMetadata{
				Resource: "Test",
				ReplacingResource: &ResourceMetadata{
					Resource: "ReplaceTest",
				},
			},
			permissions: map[string]storage.Access{"ReplaceTest": storage.Access_READ_ACCESS},
			access:      storage.Access_READ_WRITE_ACCESS,
		},
		"replaced resource with no access asking for READ access should return false": {
			resource: ResourceMetadata{
				Resource: "Test",
				ReplacingResource: &ResourceMetadata{
					Resource: "ReplaceTest",
				},
			},
			permissions: map[string]storage.Access{"Other Resource": storage.Access_READ_ACCESS},
			access:      storage.Access_READ_WRITE_ACCESS,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.expectedRes, CheckResourceForAccess(c.resource, c.permissions, c.access))
		})
	}
}
