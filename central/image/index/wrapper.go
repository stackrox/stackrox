package index

import (
	"github.com/gogo/protobuf/proto"
	imageDackBox "github.com/stackrox/rox/central/image/dackbox"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Wrapper implements the wrapper interface for use in dackbox.
type Wrapper struct{}

// Wrap wraps the the input key and msg into a indexable object with the type declared.
func (ir Wrapper) Wrap(key []byte, msg proto.Message) (string, interface{}) {
	id := imageDackBox.BucketHandler.GetID(key)
	if msg == nil {
		return id, nil
	}
	return id, &imageWrapper{
		Image: msg.(*storage.Image),
		Type:  v1.SearchCategory_IMAGES.String(),
	}
}
