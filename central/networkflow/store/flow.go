package store

import (
	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/types"
	"github.com/pborman/uuid"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/timestamp"
)

const networkFlowBucket = "networkFlows"
const networkFlowLastUpdateTSBucket = "networkFlowsLastUpdateTS"

// FlowStore stores all of the flows for a single cluster.
type FlowStore interface {
	GetAllFlows() ([]*v1.NetworkFlow, types.Timestamp, error)
	GetFlow(props *v1.NetworkFlowProperties) (*v1.NetworkFlow, error)

	UpsertFlows(flows []*v1.NetworkFlow, lastUpdateTS timestamp.MicroTS) error
	RemoveFlow(props *v1.NetworkFlowProperties) error
}

// NewFlowStore returns a new FlowStore instance for the given cluster using the provided bolt DB instance.
// If a FlowStore for the input clusterID has already been created, then it will container and modify the same
// information.
func NewFlowStore(db *bolt.DB, clusterID string) FlowStore {
	bucketName := networkFlowBucket + clusterID
	bolthelper.RegisterBucketOrPanic(db, bucketName)
	bolthelper.RegisterBucketOrPanic(db, networkFlowLastUpdateTSBucket)
	return &flowStoreImpl{
		db:                 db,
		bucketName:         bucketName,
		updateTSBucketName: networkFlowLastUpdateTSBucket,
		bucketUUID:         uuid.Parse(clusterID),
	}
}
