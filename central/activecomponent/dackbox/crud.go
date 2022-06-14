package dackbox

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/crud"
	"github.com/stackrox/stackrox/pkg/dbhelper"
	"github.com/stackrox/stackrox/pkg/features"
)

var (
	// Bucket stores the active component.
	Bucket = []byte("active_components")

	// BucketHandler is the bucket's handler.
	BucketHandler = &dbhelper.BucketHandler{BucketPrefix: Bucket}

	// Reader reads storage.ActiveComponent(s) directly from the store.
	Reader = crud.NewReader(
		crud.WithAllocFunction(alloc),
	)

	// Upserter writes storage.ActiveComponent(s) directly to the store.
	Upserter = crud.NewUpserter(
		crud.WithKeyFunction(KeyFunc),
		crud.AddToIndex(),
	)

	// Deleter deletes the storage.ActiveComponent(s) from the store.
	Deleter = crud.NewDeleter(
		crud.RemoveFromIndex(),
	)
)

func init() {
	if !features.ActiveVulnManagement.Enabled() {
		return
	}

	globaldb.RegisterBucket(Bucket, "Active Component")
}

// KeyFunc returns the key with prefix.
func KeyFunc(msg proto.Message) []byte {
	unPrefixed := []byte(msg.(*storage.ActiveComponent).GetId())
	return dbhelper.GetBucketKey(Bucket, unPrefixed)
}

func alloc() proto.Message {
	return &storage.ActiveComponent{}
}
