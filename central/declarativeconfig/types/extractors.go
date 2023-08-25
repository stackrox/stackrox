package types

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/utils"
)

// IDExtractor extracts the ID from proto messages.
type IDExtractor func(m proto.Message) string

// NameExtractor extracts the name from proto messages.
type NameExtractor func(m proto.Message) string

// UniversalIDExtractor provides a way to extract the ID from proto messages.
func UniversalIDExtractor() IDExtractor {
	return extractIDFromProtoMessage
}

// UniversalNameExtractor provides a way to extract the name from proto messages.
func UniversalNameExtractor() NameExtractor {
	return extractNameFromProtoMessage
}

func extractIDFromProtoMessage(message proto.Message) string {
	// Special case, as the group specifies the ID nested within the groups properties.
	if group, ok := message.(*storage.Group); ok {
		return group.GetProps().GetId()
	}
	// Special case, as the name of the role is the ID.
	if role, ok := message.(*storage.Role); ok {
		return role.GetName()
	}

	messageWithID, ok := message.(interface {
		GetId() string
	})
	// Theoretically, this should never happen unless we add more proto messages to the reconciliation. Hence, we use
	// utils.Should to guard this.
	if !ok {
		utils.Should(errox.InvariantViolation.Newf("could not retrieve ID from message type %T %+v",
			message, message))
		return ""
	}
	return messageWithID.GetId()
}

func extractNameFromProtoMessage(message proto.Message) string {
	// Special case, as the group specifies no name we will use a combination of multiple values to identify it.
	if group, ok := message.(*storage.Group); ok {
		return fmt.Sprintf("group %s:%s:%s for auth provider ID %s",
			group.GetProps().GetKey(), group.GetProps().GetValue(), group.GetRoleName(), group.GetProps().GetAuthProviderId())
	}

	messageWithName, ok := message.(interface {
		GetName() string
	})
	// This may happen for some resources (such as groups) as they do not define a name.
	if !ok {
		return ""
	}

	return messageWithName.GetName()
}
