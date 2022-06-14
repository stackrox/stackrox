package dackbox

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/crud"
	"github.com/stackrox/stackrox/pkg/dbhelper"
)

var (
	// Bucket is the prefix for stored nodes.
	Bucket = []byte("nodes")

	// BucketHandler is the bucket's handler.
	BucketHandler = &dbhelper.BucketHandler{BucketPrefix: Bucket}

	// Reader reads nodes.
	Reader = crud.NewReader(
		crud.WithAllocFunction(alloc),
	)

	// Upserter upserts nodes.
	Upserter = crud.NewUpserter(
		crud.WithKeyFunction(KeyFunc),
		crud.AddToIndex(),
	)

	// Deleter deletes nodes.
	Deleter = crud.NewDeleter(crud.RemoveFromIndex())
)

func init() {
	globaldb.RegisterBucket(Bucket, "Node")
}

// KeyFunc returns the key for a node object
func KeyFunc(msg proto.Message) []byte {
	unPrefixed := []byte(msg.(*storage.Node).GetId())
	return dbhelper.GetBucketKey(Bucket, unPrefixed)
}

func alloc() proto.Message {
	return &storage.Node{}
}
