package index

import (
	deploymentDackBox "github.com/stackrox/rox/central/deployment/dackbox"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/transitional/protocompat/proto"
)

// Wrapper implements the wrapper interface for use in dackbox.
type Wrapper struct{}

// Wrap wraps the the input key and msg into a indexable object with the type declared.
func (ir Wrapper) Wrap(key []byte, msg proto.Message) (string, interface{}) {
	id := deploymentDackBox.BucketHandler.GetID(key)
	if msg == nil {
		return id, nil
	}
	return id, &deploymentWrapper{
		Deployment: msg.(*storage.Deployment),
		Type:       v1.SearchCategory_DEPLOYMENTS.String(),
	}
}
