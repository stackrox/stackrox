package dackbox

import (
	"github.com/stackrox/rox/pkg/badgerhelper"
)

var (
	// Bucket is the prefix for stored namespaces.
	Bucket = []byte("namespaces")

	// BucketHandler is the bucket's handler.
	BucketHandler = &badgerhelper.BucketHandler{BucketPrefix: Bucket}
)
