package dackbox

import "github.com/stackrox/rox/pkg/badgerhelper"

var (
	// Bucket is the prefix for stored clusters.
	Bucket = []byte("clusters")

	// BucketHandler is the bucket's handler.
	BucketHandler = &badgerhelper.BucketHandler{BucketPrefix: Bucket}
)
