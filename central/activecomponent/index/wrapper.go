package index

import (
	"github.com/gogo/protobuf/proto"
	activeComponentDackBox "github.com/stackrox/stackrox/central/activecomponent/dackbox"
	"github.com/stackrox/stackrox/central/activecomponent/index/internal"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
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
		ActiveComponent: internal.ConvertToIndexContexts(msg.(*storage.ActiveComponent)),
		Type:            v1.SearchCategory_ACTIVE_COMPONENT.String(),
	}
}
