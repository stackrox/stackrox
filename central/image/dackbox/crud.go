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
	// Bucket is the prefix for image objects in the db.
	Bucket = []byte("imageBucket")
	// ListBucket is the prefix for list image objects in the db.
	ListBucket = []byte("images_list")

	// BucketHandler is the bucket's handler.
	BucketHandler = &dbhelper.BucketHandler{BucketPrefix: Bucket}

	// ListBucketHandler is the list bucket's handler.
	ListBucketHandler = &dbhelper.BucketHandler{BucketPrefix: ListBucket}

	// Reader reads images.
	Reader = crud.NewReader(
		crud.WithAllocFunction(alloc),
	)

	// ListReader reads list images from the db.
	ListReader = crud.NewReader(
		crud.WithAllocFunction(listAlloc),
	)

	// Upserter upserts images.
	Upserter = crud.NewUpserter(
		crud.WithKeyFunction(KeyFunc),
		crud.AddToIndex(),
	)

	// ListUpserter upserts a list image.
	ListUpserter = crud.NewUpserter(
		crud.WithKeyFunction(ListKeyFunc),
	)

	// Deleter deletes images and list images by id.
	Deleter = crud.NewDeleter(crud.RemoveFromIndex())

	// ListDeleter deletes a list image.
	ListDeleter = crud.NewDeleter()
)

func init() {
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		globaldb.RegisterBucket(Bucket, "Image")
		globaldb.RegisterBucket(ListBucket, "List Image")
	}
}

// KeyFunc returns the key for an image object
func KeyFunc(msg proto.Message) []byte {
	unPrefixed := []byte(msg.(interface{ GetId() string }).GetId())
	return dbhelper.GetBucketKey(Bucket, unPrefixed)
}

// ListKeyFunc returns the key for a list image.
func ListKeyFunc(msg proto.Message) []byte {
	unPrefixed := []byte(msg.(interface{ GetId() string }).GetId())
	return dbhelper.GetBucketKey(ListBucket, unPrefixed)
}

func alloc() proto.Message {
	return &storage.Image{}
}

func listAlloc() proto.Message {
	return &storage.ListImage{}
}
