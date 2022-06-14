package index

import (
	"github.com/gogo/protobuf/proto"
	edgeDackBox "github.com/stackrox/rox/central/clustercveedge/dackbox"
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
	return id, &clusterCVEEdgeWrapper{
		ClusterCVEEdge: msg.(*storage.ClusterCVEEdge),
		Type:           v1.SearchCategory_CLUSTER_VULN_EDGE.String(),
	}
}
