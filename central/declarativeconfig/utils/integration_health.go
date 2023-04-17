package utils

import (
	"fmt"
	"path"

	"github.com/gogo/protobuf/proto"
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/declarativeconfig/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/utils"
)

// IntegrationHealthForProtoMessage returns a storage.IntegrationHealth for the given proto.Message.
// The integration health will be marked as unhealthy if err != nil, and healthy if err == nil.
// Note: Handler can be left empty. In this case, the name of the integration health will be updated to not include the
// handler name.
func IntegrationHealthForProtoMessage(message proto.Message, handler string, err error, idExtractor types.IDExtractor,
	nameExtractor types.NameExtractor) *storage.IntegrationHealth {
	messageID := idExtractor(message)
	messageName := NameForIntegrationHealthFromProtoMessage(message, handler, nameExtractor, idExtractor)

	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}

	return &storage.IntegrationHealth{
		Id:            messageID,
		Name:          messageName,
		Type:          storage.IntegrationHealth_DECLARATIVE_CONFIG,
		Status:        utils.IfThenElse(err != nil, storage.IntegrationHealth_UNHEALTHY, storage.IntegrationHealth_HEALTHY),
		ErrorMessage:  errMsg,
		LastTimestamp: timestamp.TimestampNow(),
	}
}

// NameForIntegrationHealthFromProtoMessage will return the name to use within a storage.IntegrationHealth.
// This will take the name deduced from proto.Message, and optionally add the handler information as well to the name as
// well, if a handler is given. If the handler is set to empty, then simply the name deduced from the proto.Message will
// be returned.
func NameForIntegrationHealthFromProtoMessage(message proto.Message, handler string, nameExtractor types.NameExtractor,
	idExtractor types.IDExtractor) string {
	messageName := stringutils.FirstNonEmpty(nameExtractor(message), idExtractor(message))
	if handler != "" {
		messageName = fmt.Sprintf("%s in config map %s", messageName, path.Base(handler))
	}
	return messageName
}
