package dackbox

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/crud"
	"github.com/stackrox/rox/pkg/dbhelper"
)

var (
	// Bucket stores the component to vulnerability edges.
	Bucket = []byte("comp_to_vuln")

	// BucketHandler is the bucket's handler.
	BucketHandler = &dbhelper.BucketHandler{BucketPrefix: Bucket}

	// Reader reads storage.CVEs directly from the store.
	Reader = crud.NewReader(
		crud.WithAllocFunction(Alloc),
	)

	// Upserter writes storage.CVEs directly to the store.
	Upserter = crud.NewUpserter(
		crud.WithKeyFunction(KeyFunc),
		crud.AddToIndex(),
	)

	// Deleter deletes vulns from the store.
	Deleter = crud.NewDeleter(
		crud.Shared(),
		crud.RemoveFromIndex(),
	)
)

func init() {
	globaldb.RegisterBucket(Bucket, "Component Vuln Edge")
}

// KeyFunc returns the key for a ComponentCVEEdge.
func KeyFunc(msg proto.Message) []byte {
	unPrefixed := []byte(msg.(interface{ GetId() string }).GetId())
	return dbhelper.GetBucketKey(Bucket, unPrefixed)
}

// Alloc allocates a ComponentCVEEdge.
func Alloc() proto.Message {
	return &storage.ComponentCVEEdge{}
}
