package processtagsstore

import (
	"github.com/stackrox/rox/central/analystnotes"
	"github.com/stackrox/rox/pkg/bolthelper"
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
}

// New returns a new, ready-to-use, store.
func New(db *bbolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, processTagsBucket)
	return &storeImpl{
		bucketRef: bolthelper.TopLevelRef(db, processTagsBucket),
	}
}
