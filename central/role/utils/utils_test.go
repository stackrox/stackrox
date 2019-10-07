package utils

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestFillAccessList(t *testing.T) {
	testRole := &storage.Role{
		GlobalAccess: storage.Access_READ_WRITE_ACCESS,
		ResourceToAccess: map[string]storage.Access{
			"Alert": storage.Access_READ_ACCESS,
		},
	}

	FillAccessList(testRole)
	assert.Equal(t, testRole.GetResourceToAccess()["Alert"], storage.Access_READ_WRITE_ACCESS)
}
