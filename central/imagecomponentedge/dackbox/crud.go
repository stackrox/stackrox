package dackbox

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/dackbox/crud"
)

var (
	// Bucket stores the image to component edges.
	Bucket = []byte("image_to_comp")

	// BucketHandler is the bucket's handler.
	BucketHandler = &badgerhelper.BucketHandler{BucketPrefix: Bucket}

	// Reader reads storage.ImageComponentEdges directly from the store.
	Reader = crud.NewReader(
		crud.WithAllocFunction(alloc),
	)

	// Upserter writes storage.ImageComponentEdges directly to the store.
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
	globaldb.RegisterBucket(Bucket, "Image Component Edge")
}

func keyFunc(msg proto.Message) []byte {
	return []byte(msg.(interface{ GetId() string }).GetId())
}

func alloc() proto.Message {
	return &storage.ImageComponentEdge{}
}
