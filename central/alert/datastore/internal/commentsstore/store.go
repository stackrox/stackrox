package commentsstore

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	bolt "go.etcd.io/bbolt"
)

var (
	alertCommentsBucket = []byte("alertComments")
)

// Store provides storage functionality for alert comments.
//go:generate mockgen-wrapper
type Store interface {
	GetCommentsForAlert(alertID string) ([]*storage.Comment, error)
	GetComment(alertID, commentID string) (*storage.Comment, error)
	// AddAlertComment adds a comment to the store, and returns the assigned id.
	// Note that the passed in object is modified by the store to add a
	// createdAt and lastModified timestamp.
	AddAlertComment(comment *storage.Comment) (string, error)
	// UpdateAlertComment updates an existing alert comment.
	// Note that the passed in object is modified by the store to add a
	// createdAt and lastModified timestamp.
	UpdateAlertComment(comment *storage.Comment) error
	RemoveAlertComment(alertID, commentID string) error
	RemoveAlertComments(alertID string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, alertCommentsBucket)
	return &storeImpl{
		DB: db,
	}
}
