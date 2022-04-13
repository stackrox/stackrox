package dackbox

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/crud"
	"github.com/stackrox/stackrox/pkg/dbhelper"
)

var (
	// Bucket stores the node to cve edges.
	Bucket = []byte("node_to_cve")

	// BucketHandler is the bucket's handler.
	BucketHandler = &dbhelper.BucketHandler{BucketPrefix: Bucket}

	// Reader reads storage.NodeCVEEdges directly from the store.
	Reader = crud.NewReader(
		crud.WithAllocFunction(alloc),
	)

	// Upserter writes storage.NodeCVEEdges directly to the store.
	Upserter = crud.NewUpserter(
		crud.WithKeyFunction(crud.PrefixKey(Bucket, keyFunc)),
	)

	// Deleter deletes the edges from the store.
	Deleter = crud.NewDeleter(
		crud.Shared(),
	)
)

func init() {
	globaldb.RegisterBucket(Bucket, "Node CVE Edge")
}

func keyFunc(msg proto.Message) []byte {
	return []byte(msg.(*storage.NodeCVEEdge).GetId())
}

func alloc() proto.Message {
	return &storage.NodeCVEEdge{}
}
