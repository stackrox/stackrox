package utils

import (
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestResourceTypeFromProtoMessage(t *testing.T) {
	cases := []struct {
		msg          proto.Message
		resourceType storage.DeclarativeConfigHealth_ResourceType
	}{
		{
			msg:          &storage.AuthProvider{},
			resourceType: storage.DeclarativeConfigHealth_AUTH_PROVIDER,
		},
		{
			msg:          &storage.SimpleAccessScope{},
			resourceType: storage.DeclarativeConfigHealth_ACCESS_SCOPE,
		},
		{
			msg:          &storage.PermissionSet{},
			resourceType: storage.DeclarativeConfigHealth_PERMISSION_SET,
		},
		{
			msg:          &storage.Role{},
			resourceType: storage.DeclarativeConfigHealth_ROLE,
		},
		{
			msg:          &storage.Group{},
			resourceType: storage.DeclarativeConfigHealth_GROUP,
		},
		{
			msg:          &storage.Notifier{},
			resourceType: storage.DeclarativeConfigHealth_NOTIFIER,
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("msg %v", c.msg), func(t *testing.T) {
			assert.Equal(t, c.resourceType, resourceTypeFromProtoMessage(c.msg))
		})
	}
}
