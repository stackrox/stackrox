package protoutils

import (
	"github.com/golang/protobuf/proto"
)

type protoMessageWithID interface {
	proto.Message
	GetId() string
}

// GetIDs returns the IDs from the given messages.
func GetIDs[T protoMessageWithID](messages []T) []string {
	ids := make([]string, 0, len(messages))
	for _, msg := range messages {
		ids = append(ids, msg.GetId())
	}
	return ids
}
