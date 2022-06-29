package index

import (
	"github.com/gogo/protobuf/proto"
	activeComponentDackBox "github.com/stackrox/rox/central/activecomponent/dackbox"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Wrapper implements the wrapper interface for use in dackbox.
type Wrapper struct{}

// Wrap wraps the input key and msg into an indexable object with the type declared.
func (ir Wrapper) Wrap(key []byte, msg proto.Message) (string, interface{}) {
	id := activeComponentDackBox.BucketHandler.GetID(key)
	if msg == nil {
		return id, nil
	}

	return id, &activeComponentWrapper{
		ActiveComponent: msg.(*storage.ActiveComponent),
		Type:            v1.SearchCategory_ACTIVE_COMPONENT.String(),
	}
}
