package commentsstore

import (
	"github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/central/analystnotes"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

var (
	processCommentsBucket = []byte("process_comments")
)

// Store stores process comments.
//go:generate mockgen-wrapper
type Store interface {
	AddProcessComment(key *analystnotes.ProcessNoteKey, comment *storage.Comment) (string, error)
	UpdateProcessComment(key *analystnotes.ProcessNoteKey, comment *storage.Comment) error

	GetComment(key *analystnotes.ProcessNoteKey, commentID string) (*storage.Comment, error)
	GetCommentsForProcessKey(key *analystnotes.ProcessNoteKey) ([]*storage.Comment, error)

	RemoveProcessComment(key *analystnotes.ProcessNoteKey, commentID string) error
	RemoveAllProcessComments(key *analystnotes.ProcessNoteKey) error
}

// New returns a new, ready-to-use, store.
func New(db *bbolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, processCommentsBucket)
	return &storeImpl{
		bucketRef: bolthelper.TopLevelRef(db, processCommentsBucket),
	}
}
