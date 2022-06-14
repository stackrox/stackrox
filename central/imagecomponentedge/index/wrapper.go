package index

import (
	"github.com/gogo/protobuf/proto"
	edgeDackBox "github.com/stackrox/rox/central/imagecomponentedge/dackbox"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Wrapper implements the wrapper interface for use in dackbox.
type Wrapper struct{}

// Wrap wraps the the input key and msg into a indexable object with the type declared.
func (ir Wrapper) Wrap(key []byte, msg proto.Message) (string, interface{}) {
	id := edgeDackBox.BucketHandler.GetID(key)
	if msg == nil {
		return id, nil
	}
	return id, &imageComponentEdgeWrapper{
		ImageComponentEdge: msg.(*storage.ImageComponentEdge),
		Type:               v1.SearchCategory_IMAGE_COMPONENT_EDGE.String(),
	}
}
