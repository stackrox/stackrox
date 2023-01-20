package declarativeconfig

import (
	"testing"

	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

func TestDeterministicUUIDs(t *testing.T) {
	declarativeUUIDFuncs := []func(string) uuid.UUID{
		NewDeclarativeAuthProviderUUID,
		NewDeclarativeGroupUUID,
		NewDeclarativePermissionSetUUID,
		NewDeclarativeAccessScopeUUID,
	}
	dummyName := "dummy-resource"

	// 1. Test that using the same name will lead to the same UUID being created.
	for _, f := range declarativeUUIDFuncs {
		firstID := f(dummyName)
		secondID := f(dummyName)
		assert.Equal(t, firstID, secondID)
	}

	// 2. Test that using the same name won't lead to clashes within the different namespaces.
	for i, f := range declarativeUUIDFuncs {
		id := f(dummyName)
		for j, f := range declarativeUUIDFuncs {
			if j == i {
				continue
			}
			otherID := f(dummyName)
			assert.NotEqual(t, id, otherID)
		}
	}
}
