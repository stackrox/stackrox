package store

import (
	"github.com/boltdb/bolt"
	"github.com/pborman/uuid"
	"github.com/stackrox/rox/generated/data"
	"github.com/stackrox/rox/pkg/bolthelper"
)

const networkFlowBucket = "networkFlows"

// FlowStore stores all of the flows for a single cluster.
type FlowStore interface {
	GetAllFlows() ([]*data.NetworkFlow, error)
	GetFlow(props *data.NetworkFlowProperties) (*data.NetworkFlow, error)

	AddFlow(flow *data.NetworkFlow) error
	UpdateFlow(flow *data.NetworkFlow) error
	UpsertFlow(flow *data.NetworkFlow) error
	RemoveFlow(props *data.NetworkFlowProperties) error
}

// NewFlowStore returns a new FlowStore instance for the given cluster using the provided bolt DB instance.
// If a FlowStore for the input clusterID has already been created, then it will container and modify the same
// information.
func NewFlowStore(db *bolt.DB, clusterID string) FlowStore {
	bucketName := networkFlowBucket + clusterID
	bolthelper.RegisterBucketOrPanic(db, bucketName)
	return &flowStoreImpl{
		db:         db,
		bucketName: bucketName,
		bucketUUID: uuid.Parse(clusterID),
	}
}
