package utils

import (
	"fmt"
	"path"
	"reflect"

	"github.com/stackrox/rox/central/declarativeconfig/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	roleHealthIDNS = "role-config-health"
)

var (
	protoMessageToHealthResourceTypeMap = map[reflect.Type]storage.DeclarativeConfigHealth_ResourceType{
		types.AuthProviderType:               storage.DeclarativeConfigHealth_AUTH_PROVIDER,
		types.AccessScopeType:                storage.DeclarativeConfigHealth_ACCESS_SCOPE,
		types.GroupType:                      storage.DeclarativeConfigHealth_GROUP,
		types.PermissionSetType:              storage.DeclarativeConfigHealth_PERMISSION_SET,
		types.RoleType:                       storage.DeclarativeConfigHealth_ROLE,
		types.NotifierType:                   storage.DeclarativeConfigHealth_NOTIFIER,
		types.AuthMachineToMachineConfigType: storage.DeclarativeConfigHealth_AUTH_MACHINE_TO_MACHINE_CONFIG,
	}
)

// HealthStatusForProtoMessage returns a storage.DeclarativeConfigHealth for the given protocompat.Message.
// The health will be marked as unhealthy if err != nil, and healthy if err == nil.
// Note: Handler can be left empty. In this case, the name of the health will be updated to not include the
// handler name.
func HealthStatusForProtoMessage(message protocompat.Message, handler string, err error, idExtractor types.IDExtractor,
	nameExtractor types.NameExtractor) *storage.DeclarativeConfigHealth {
	messageID := idExtractor(message)
	messageName := resourceNameFromProtoMessage(message, nameExtractor, idExtractor)

	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}

	name := resourceNameFromProtoMessage(message, nameExtractor, idExtractor)

	resourceType := resourceTypeFromProtoMessage(message)

	// Special case: the store itself requires UUIDs.
	// For roles, we currently do not have any ID, but their name.
	// Hence, we need to create a UUID on-the-fly for it.
	if resourceType == storage.DeclarativeConfigHealth_ROLE {
		messageID = HealthStatusIDForRole(messageID)
	}

	return &storage.DeclarativeConfigHealth{
		Id: messageID,
		Name: utils.IfThenElse(handler != "",
			fmt.Sprintf("%s in config map %s", messageName, path.Base(handler)),
			name),
		ResourceName: name,
		ResourceType: resourceType,
		Status: utils.IfThenElse(err != nil, storage.DeclarativeConfigHealth_UNHEALTHY,
			storage.DeclarativeConfigHealth_HEALTHY),
		ErrorMessage:  errMsg,
		LastTimestamp: protocompat.TimestampNow(),
	}
}

// HealthStatusForHandler will create a storage.DeclarativeConfigHealth object for a handler.
// The health will be marked as unhealthy if err != nil, and health if err == nil.
func HealthStatusForHandler(handlerID string, err error) *storage.DeclarativeConfigHealth {
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}

	return &storage.DeclarativeConfigHealth{
		Id:           declarativeconfig.NewDeclarativeHandlerUUID(path.Base(handlerID)).String(),
		Name:         fmt.Sprintf("Config Map %s", path.Base(handlerID)),
		ResourceName: path.Base(handlerID),
		ResourceType: storage.DeclarativeConfigHealth_CONFIG_MAP,
		Status: utils.IfThenElse(err != nil, storage.DeclarativeConfigHealth_UNHEALTHY,
			storage.DeclarativeConfigHealth_HEALTHY),
		ErrorMessage:  errMsg,
		LastTimestamp: protocompat.TimestampNow(),
	}
}

// resourceNameFromProtoMessage will return the resource name to use within a storage.DeclarativeConfigHealth.
// This will take the name deduced from protocompat.Message, which will either be the name or if no name is given its ID.
func resourceNameFromProtoMessage(message protocompat.Message, nameExtractor types.NameExtractor,
	idExtractor types.IDExtractor) string {
	messageName := stringutils.FirstNonEmpty(nameExtractor(message), idExtractor(message))
	return messageName
}

func resourceTypeFromProtoMessage(message protocompat.Message) storage.DeclarativeConfigHealth_ResourceType {
	msgType := reflect.TypeOf(message)
	healthType, found := protoMessageToHealthResourceTypeMap[msgType]
	if !found {
		utils.Must(errox.InvariantViolation.Newf("unsupported type given for proto message %+v, "+
			"returning the default type", reflect.TypeOf(message)))
		// Still return here although we will panic above.
		return 0
	}
	return healthType
}

// HealthStatusIDForRole returns a UUID for the health status based on the role's name.
// The UUID is deterministic.
func HealthStatusIDForRole(name string) string {
	return uuid.NewV5FromNonUUIDs(roleHealthIDNS, name).String()
}
