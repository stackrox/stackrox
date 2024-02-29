// This file was originally generated with
// //go:generate cp ../../../central/node/dackbox/crud.go node/crud.go
package dackbox

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/crud"
	"github.com/stackrox/rox/pkg/dbhelper"
	"github.com/stackrox/rox/pkg/protocompat"
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

// KeyFunc returns the key for a node object
func KeyFunc(msg protocompat.Message) []byte {
	unPrefixed := []byte(msg.(*storage.Node).GetId())
	return dbhelper.GetBucketKey(Bucket, unPrefixed)
}

func alloc() protocompat.Message {
	return &storage.Node{}
}
