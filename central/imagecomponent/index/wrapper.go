package index

import (
	"github.com/gogo/protobuf/proto"
	componentDackBox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Wrapper implements the wrapper interface for use in dackbox.
type Wrapper struct{}

// Wrap wraps the the input key and msg into a indexable object with the type declared.
func (ir Wrapper) Wrap(key []byte, msg proto.Message) (string, interface{}) {
	id := componentDackBox.BucketHandler.GetID(key)
	if msg == nil {
		return id, nil
	}
	return id, &imageComponentWrapper{
		ImageComponent: msg.(*storage.ImageComponent),
		Type:           v1.SearchCategory_IMAGE_COMPONENTS.String(),
	}
}
