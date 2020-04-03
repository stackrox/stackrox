package commentsstore

import (
	"strconv"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

type storeImpl struct {
	*bolt.DB
}

func (b *storeImpl) GetCommentsForAlert(alertID string) ([]*storage.Comment, error) {
	var comments []*storage.Comment
	err := b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(alertCommentsBucket)
		alertIDBucket := bucket.Bucket([]byte(alertID))
		if alertIDBucket == nil {
			return nil
		}
		return alertIDBucket.ForEach(func(k, v []byte) error {
			var comment storage.Comment
			if err := proto.Unmarshal(v, &comment); err != nil {
				return errors.Wrapf(err, "unmarshalling comment with id: %q", comment.GetCommentId())
			}
			comments = append(comments, &comment)
			return nil
		})
	})
	return comments, err
}

func (b *storeImpl) GetComment(alertID string, commentID string) (*storage.Comment, error) {
	var comment *storage.Comment
	err := b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(alertCommentsBucket)
		alertIDBucket := bucket.Bucket([]byte(alertID))
		if alertIDBucket == nil {
			return errors.Errorf("alert id %q does not have any comments", alertID)
		}
		bytes := alertIDBucket.Get([]byte(commentID))
		if bytes == nil {
			return nil
		}
		comment = new(storage.Comment)
		if err := proto.Unmarshal(bytes, comment); err != nil {
			return errors.Wrapf(err, "unmarshalling comment with id: %q", commentID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return comment, nil
}

// addStandardFields populates the fields in the comment that the store is responsible for populating.
func addStandardFields(comment *storage.Comment, lastModified *types.Timestamp) {
	comment.ResourceType = storage.ResourceType_ALERT
	comment.LastModified = lastModified
}

func (b *storeImpl) AddAlertComment(comment *storage.Comment) (string, error) {
	if comment == nil {
		return "", errors.New("cannot add a nil comment")
	}
	if comment.GetCommentId() != "" {
		return "", errors.Errorf("cannot add a comment that has already been assigned an id: %q", comment.GetCommentId())
	}
	err := b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(alertCommentsBucket)

		_, err := bucket.CreateBucketIfNotExists([]byte(comment.GetResourceId()))
		if err != nil {
			return errors.Wrap(err, "creating sub-bucket")
		}
		alertIDBucket := bucket.Bucket([]byte(comment.GetResourceId()))
		id, err := alertIDBucket.NextSequence()
		if err != nil {
			return errors.Wrapf(err, "getting next sequence for alertIDBucket with id: %q", comment.GetResourceId())
		}
		commentID := strconv.FormatUint(id, 10)
		comment.CommentId = commentID
		currTime := types.TimestampNow()
		comment.CreatedAt = currTime
		addStandardFields(comment, currTime)
		bytes, err := proto.Marshal(comment)
		if err != nil {
			return errors.Wrapf(err, "marshalling comment with id: %q", comment.GetCommentId())
		}
		return alertIDBucket.Put([]byte(commentID), bytes)
	})
	if err != nil {
		return "", err
	}
	return comment.GetCommentId(), nil
}

func (b *storeImpl) UpdateAlertComment(comment *storage.Comment) error {
	if comment == nil {
		return errors.New("cannot edit a nil comment")
	}
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(alertCommentsBucket)
		alertIDBucket := bucket.Bucket([]byte(comment.GetResourceId()))
		if alertIDBucket == nil {
			return errors.Errorf("alert id %q does not have any comments", comment.GetResourceId())
		}
		bytes := alertIDBucket.Get([]byte(comment.GetCommentId()))
		if bytes == nil {
			return errors.Errorf("couldn't edit nonexistent comment with id : %q", comment.GetCommentId())
		}
		var oldComment storage.Comment
		if err := proto.Unmarshal(bytes, &oldComment); err != nil {
			return errors.Wrapf(err, "unmarshalling comment with id: %q", comment.GetCommentId())
		}
		comment.CreatedAt = oldComment.GetCreatedAt()
		addStandardFields(comment, types.TimestampNow())

		bytes, err := proto.Marshal(comment)
		if err != nil {
			return errors.Wrapf(err, "marshalling comment with id: %q", comment.GetCommentId())
		}
		return alertIDBucket.Put([]byte(comment.GetCommentId()), bytes)
	})
}

func (b *storeImpl) RemoveAlertComment(alertID, commentID string) error {
	if commentID == "" {
		return errors.New("no comment id provided")

	}
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(alertCommentsBucket)
		alertIDBytes := []byte(alertID)
		alertIDBucket := bucket.Bucket(alertIDBytes)
		if alertIDBucket == nil {
			return nil
		}
		err := alertIDBucket.Delete([]byte(commentID))
		if err != nil {
			return errors.Wrapf(err, "deleting alert comment with id %q", commentID)
		}
		c := alertIDBucket.Cursor()
		firstKey, _ := c.First()
		if firstKey == nil {
			err = bucket.DeleteBucket(alertIDBytes)
			if err != nil {
				return errors.Wrapf(err, "deleting alert bucket with id %q", alertID)
			}
		}
		return nil
	})
}

func (b *storeImpl) RemoveAlertComments(alertID string) error {
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(alertCommentsBucket)
		err := bucket.DeleteBucket([]byte(alertID))
		if err != nil {
			return errors.Wrapf(err, "deleting comments of alert %q", alertID)
		}
		return nil
	})
}
