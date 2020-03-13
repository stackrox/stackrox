package commentsstore

import (
	"github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/central/comments"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

var (
	processCommentsBucket = []byte("process_comments")
)

// Store stores process comments.
//go:generate mockgen-wrapper
type Store interface {
	AddProcessComment(key *comments.ProcessCommentKey, comment *storage.Comment) (string, error)
	UpdateProcessComment(key *comments.ProcessCommentKey, comment *storage.Comment) error

	GetComment(key *comments.ProcessCommentKey, commentID string) (*storage.Comment, error)
	GetCommentsForProcessKey(key *comments.ProcessCommentKey) ([]*storage.Comment, error)

	RemoveProcessComment(key *comments.ProcessCommentKey, commentID string) error
	RemoveAllProcessComments(key *comments.ProcessCommentKey) error
}

// New returns a new, ready-to-use, store.
func New(db *bbolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, processCommentsBucket)
	return &storeImpl{
		bucketRef: bolthelper.TopLevelRef(db, processCommentsBucket),
	}
}
