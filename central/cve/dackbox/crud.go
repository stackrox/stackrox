package dackbox

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/crud"
	"github.com/stackrox/stackrox/pkg/dbhelper"
)

var (
	// Bucket stores the child image vulnerabilities.
	Bucket = []byte("image_vuln")

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
	globaldb.RegisterBucket(Bucket, "Vuln")
}

// KeyFunc returns the key for a CVE.
func KeyFunc(msg proto.Message) []byte {
	unPrefixed := []byte(msg.(interface{ GetId() string }).GetId())
	return dbhelper.GetBucketKey(Bucket, unPrefixed)
}

// Alloc allocates a CVE.
func Alloc() proto.Message {
	return &storage.CVE{}
}
