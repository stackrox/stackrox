package commentsstore

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

var (
	alertCommentsBucket = []byte("alertComments")
)

// Store provides storage functionality for alert comments.
//go:generate mockgen-wrapper
type Store interface {
	GetCommentsForAlert(alertID string) ([]*storage.Comment, error)
	AddAlertComment(comment *storage.Comment) (string, error)
	UpdateAlertComment(comment *storage.Comment) error
	RemoveAlertComment(comment *storage.Comment) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, alertCommentsBucket)
	return &storeImpl{
		DB: db,
	}
}
