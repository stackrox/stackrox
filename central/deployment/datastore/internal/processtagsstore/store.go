package processtagsstore

import (
	"github.com/stackrox/stackrox/central/analystnotes"
	"github.com/stackrox/stackrox/pkg/bolthelper"
	"go.etcd.io/bbolt"
)

var (
	processTagsBucket = []byte("process_tags")
)

// Store stores process tags.
//go:generate mockgen-wrapper
type Store interface {
	GetTagsForProcessKey(key *analystnotes.ProcessNoteKey) ([]string, error)
	UpsertProcessTags(key *analystnotes.ProcessNoteKey, tags []string) error
	RemoveProcessTags(key *analystnotes.ProcessNoteKey, tags []string) error
	// WalkTagsForDeployment walks all the tags under the given deployment,
	// and calls the passed func on it.
	// The function is only called once per unique tag.
	// If the func returns false, then execution stops early.
	WalkTagsForDeployment(deploymentID string, f func(tag string) bool) error
}

// New returns a new, ready-to-use, store.
func New(db *bbolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, processTagsBucket)
	return &storeImpl{
		bucketRef: bolthelper.TopLevelRef(db, processTagsBucket),
	}
}
