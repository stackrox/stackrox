package declarativeconfig

import (
	"testing"

	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

func TestDeterministicUUIDs(t *testing.T) {
	declarativeUUIDFuncs := map[string]func(string) uuid.UUID{
		authProviderUUIDNS:  NewDeclarativeAuthProviderUUID,
		groupUUIDNS:         NewDeclarativeGroupUUID,
		permissionSetUUIDNS: NewDeclarativePermissionSetUUID,
		accessScopeUUIDNS:   NewDeclarativeAccessScopeUUID,
		notifierUUIDNS:      NewDeclarativeNotifierUUID,
		handlerUUIDNS:       NewDeclarativeHandlerUUID,
		authM2MConfigUUIDNS: NewDeclarativeM2MAuthConfigUUID,
	}
	dummyName := "dummy-resource"

	for typeName, f := range declarativeUUIDFuncs {
		t.Run(typeName, func(it *testing.T) {
			// 1. Test that using the same name will lead to the same UUID being created.
			firstID := f(dummyName)
			secondID := f(dummyName)
			assert.Equal(it, firstID, secondID)
			firstIDString := firstID.String()
			secondIDString := secondID.String()
			assert.Equal(it, firstIDString, secondIDString)

			// 2. Test that using the same name won't lead to clashes within the different namespaces.
			for otherTypeName, otherUUIDFunc := range declarativeUUIDFuncs {
				if typeName == otherTypeName {
					continue
				}
				otherID := otherUUIDFunc(dummyName)
				assert.NotEqual(it, firstID, otherID)
				otherIDString := otherID.String()
				assert.NotEqual(it, firstIDString, otherIDString)
			}

			// 3. Test that using different names in the same namespace will lead to different UUIDs being created.
			anotherDummyName := "another-dummy-resource"
			anotherID := f(anotherDummyName)
			assert.NotEqual(it, firstID, anotherID)
			anotherIDString := anotherID.String()
			assert.NotEqual(it, firstIDString, anotherIDString)
		})
	}
}
