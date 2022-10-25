package index

import (
	edgeDackBox "github.com/stackrox/rox/central/nodecomponentedge/dackbox"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/transitional/protocompat/proto"
)

// Wrapper implements the wrapper interface for use in dackbox.
type Wrapper struct{}

// Wrap wraps the the input key and msg into a indexable object with the type declared.
func (ir Wrapper) Wrap(key []byte, msg proto.Message) (string, interface{}) {
	id := edgeDackBox.BucketHandler.GetID(key)
	if msg == nil {
		return id, nil
	}
	return id, &nodeComponentEdgeWrapper{
		NodeComponentEdge: msg.(*storage.NodeComponentEdge),
		Type:              v1.SearchCategory_NODE_COMPONENT_EDGE.String(),
	}
}
