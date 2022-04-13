package dackbox

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/crud"
	"github.com/stackrox/stackrox/pkg/dbhelper"
)

var (
	// Bucket stores the node to component edges.
	Bucket = []byte("node_to_comp")

	// BucketHandler is the bucket's handler.
	BucketHandler = &dbhelper.BucketHandler{BucketPrefix: Bucket}

	// Reader reads storage.NodeComponentEdges directly from the store.
	Reader = crud.NewReader(
		crud.WithAllocFunction(alloc),
	)

	// Upserter writes storage.NodeComponentEdges directly to the store.
	Upserter = crud.NewUpserter(
		crud.WithKeyFunction(crud.PrefixKey(Bucket, keyFunc)),
		crud.AddToIndex(),
	)

	// Deleter deletes the edges from the store.
	Deleter = crud.NewDeleter(
		crud.Shared(),
		crud.RemoveFromIndex(),
	)
)

func init() {
	globaldb.RegisterBucket(Bucket, "Node Component Edge")
}

func keyFunc(msg proto.Message) []byte {
	return []byte(msg.(*storage.NodeComponentEdge).GetId())
}

func alloc() proto.Message {
	return &storage.NodeComponentEdge{}
}
