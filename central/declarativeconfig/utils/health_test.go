package utils

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/declarativeconfig/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
)

func TestResourceTypeFromProtoMessage(t *testing.T) {
	cases := []struct {
		msg          protocompat.Message
		resourceType storage.DeclarativeConfigHealth_ResourceType
	}{
		{
			msg:          &storage.AuthMachineToMachineConfig{},
			resourceType: storage.DeclarativeConfigHealth_AUTH_MACHINE_TO_MACHINE_CONFIG,
		},
		{
			msg:          &storage.AuthProvider{},
			resourceType: storage.DeclarativeConfigHealth_AUTH_PROVIDER,
		},
		{
			msg:          &storage.Group{},
			resourceType: storage.DeclarativeConfigHealth_GROUP,
		},
		{
			msg:          &storage.Notifier{},
			resourceType: storage.DeclarativeConfigHealth_NOTIFIER,
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
			msg:          &storage.SimpleAccessScope{},
			resourceType: storage.DeclarativeConfigHealth_ACCESS_SCOPE,
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("msg type %T", c.msg), func(t *testing.T) {
			assert.Equal(t, c.resourceType, resourceTypeFromProtoMessage(c.msg))
		})
	}

	testAlertMsg := &storage.Alert{}
	t.Run(fmt.Sprintf("msg type %T", testAlertMsg), func(t *testing.T) {
		assert.Panics(t, func() {
			_ = resourceTypeFromProtoMessage(testAlertMsg)
		})
	})
}

func TestAllSupportedProtobufTypesHaveHealthTypeAssociated(t *testing.T) {
	supportedTypes := types.GetSupportedProtobufTypesInProcessingOrder()

	assert.Len(t, protoMessageToHealthResourceTypeMap, len(supportedTypes))
	for _, msgType := range supportedTypes {
		_, found := protoMessageToHealthResourceTypeMap[msgType]
		assert.True(t, found)
	}

	healthTypeCounts := make(map[storage.DeclarativeConfigHealth_ResourceType]int)
	for _, healthType := range protoMessageToHealthResourceTypeMap {
		healthTypeCounts[healthType]++
	}
	for healthType, count := range healthTypeCounts {
		assert.Equalf(t, 1, count,
			"Expected only one protobuf type with Health type %s, got %d",
			healthType.String(), count)
	}
}
