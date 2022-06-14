package index

import (
	"github.com/gogo/protobuf/proto"
	edgeDackBox "github.com/stackrox/stackrox/central/imagecveedge/dackbox"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
)

// Wrapper implements the wrapper interface for use in dackbox.
type Wrapper struct{}

// Wrap wraps the the input key and msg into a indexable object with the type declared.
func (ir Wrapper) Wrap(key []byte, msg proto.Message) (string, interface{}) {
	id := edgeDackBox.BucketHandler.GetID(key)
	if msg == nil {
		return id, nil
	}
	return id, &imageCVEEdgeWrapper{
		ImageCVEEdge: msg.(*storage.ImageCVEEdge),
		Type:         v1.SearchCategory_IMAGE_VULN_EDGE.String(),
	}
}
