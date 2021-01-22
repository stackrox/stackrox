package index

import (
	"github.com/gogo/protobuf/proto"
	nodeDackBox "github.com/stackrox/rox/central/node/dackbox"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Wrapper implements the wrapper interface for use in dackbox.
type Wrapper struct{}

// Wrap wraps the the input key and msg into a indexable object with the type declared.
func (ir Wrapper) Wrap(key []byte, msg proto.Message) (string, interface{}) {
	id := nodeDackBox.BucketHandler.GetID(key)
	if msg == nil {
		return id, nil
	}
	return id, &nodeWrapper{
		Node: msg.(*storage.Node),
		Type: v1.SearchCategory_NODES.String(),
	}
}
