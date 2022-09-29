package dackbox

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/crud"
	"github.com/stackrox/rox/pkg/dbhelper"
	"github.com/stackrox/rox/pkg/env"
)

var (
	// Bucket is the prefix for stored deployments.
	Bucket = []byte("deployments")

	// ListBucket is the prefix for stored list deployments.
	ListBucket = []byte("deployments_list")

	// BucketHandler is the bucket's handler.
	BucketHandler = &dbhelper.BucketHandler{BucketPrefix: Bucket}

	// ListBucketHandler is the list bucket's handler.
	ListBucketHandler = &dbhelper.BucketHandler{BucketPrefix: ListBucket}

	// Reader reads storage.Deployments directly from the store.
	Reader = crud.NewReader(
		crud.WithAllocFunction(Alloc),
	)

	// Upserter writes storage.Deployments.
	Upserter = crud.NewUpserter(
		crud.WithKeyFunction(KeyFunc),
		crud.AddToIndex(),
	)

	// ListReader reads ListDeployments from the DB.
	ListReader = crud.NewReader(
		crud.WithAllocFunction(ListAlloc),
	)

	// ListUpserter writes storage.ListDeployments.
	ListUpserter = crud.NewUpserter(
		crud.WithKeyFunction(ListKeyFunc),
	)

	// Deleter deletes deployments from the store.
	Deleter = crud.NewDeleter(crud.RemoveFromIndex())

	// ListDeleter deletes list deployments from the store.
	ListDeleter = crud.NewDeleter()
)

// Alloc allocates a new deployment.
func Alloc() proto.Message {
	return &storage.Deployment{}
}

// ListAlloc allocates a new list deployment.
func ListAlloc() proto.Message {
	return &storage.ListDeployment{}
}

func init() {
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		globaldb.RegisterBucket(Bucket, "Deployment")
		globaldb.RegisterBucket(ListBucket, "List Deployment")
	}
}

// KeyFunc returns the key for a deployment.
func KeyFunc(msg proto.Message) []byte {
	unPrefixed := []byte(msg.(interface{ GetId() string }).GetId())
	return dbhelper.GetBucketKey(Bucket, unPrefixed)
}

// ListKeyFunc returns the key for a list deployment.
func ListKeyFunc(msg proto.Message) []byte {
	unPrefixed := []byte(msg.(interface{ GetId() string }).GetId())
	return dbhelper.GetBucketKey(ListBucket, unPrefixed)
}
