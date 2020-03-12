package dackbox

import (
	"github.com/stackrox/rox/pkg/badgerhelper"
)

var (
	// SACBucket is the prefix for namespace names used for SAC filtering.
	SACBucket = []byte("namespacesSACBucket")
	// Bucket is the prefix for stored namespaces.
	Bucket = []byte("namespaces")

	// BucketHandler is the bucket's handler.
	BucketHandler = &badgerhelper.BucketHandler{BucketPrefix: Bucket}
	// SACBucketHandler is the SAC bucket's handler.
	SACBucketHandler = &badgerhelper.BucketHandler{BucketPrefix: SACBucket}
)
