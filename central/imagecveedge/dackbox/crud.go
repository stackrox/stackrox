package dackbox

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/crud"
	"github.com/stackrox/rox/pkg/dbhelper"
)

var (
	// Bucket stores the image to cve edges.
	Bucket = []byte("image_to_cve")

	// BucketHandler is the bucket's handler.
	BucketHandler = &dbhelper.BucketHandler{BucketPrefix: Bucket}

	// Reader reads storage.ImageCVEEdges directly from the store.
	Reader = crud.NewReader(
		crud.WithAllocFunction(alloc),
	)

	// Upserter writes storage.ImageCVEEdges directly to the store.
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
	globaldb.RegisterBucket(Bucket, "Image CVE Edge")
}

func keyFunc(msg proto.Message) []byte {
	return []byte(msg.(*storage.ImageCVEEdge).GetId())
}

func alloc() proto.Message {
	return &storage.ImageCVEEdge{}
}
