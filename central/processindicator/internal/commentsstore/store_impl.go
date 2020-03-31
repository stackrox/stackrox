package commentsstore

import (
	"github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/analystnotes"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/binenc"
	"github.com/stackrox/rox/pkg/bolthelper"
)

type storeImpl struct {
	bucketRef bolthelper.BucketRef
}

func addStandardFields(comment *storage.Comment, lastModified *types.Timestamp) {
	comment.ResourceType = storage.ResourceType_PROCESS
	comment.LastModified = lastModified
}

func validateKeyAndComment(key *analystnotes.ProcessNoteKey, comment *storage.Comment) error {
	if err := key.Validate(); err != nil {
		return err
	}

	if comment == nil {
		return errors.New("invalid comment: was nil")
	}

	return nil
}

func getOrCreateProcessSubBucket(b *bbolt.Bucket, key *analystnotes.ProcessNoteKey) (*bbolt.Bucket, error) {
	deploymentSubBucket, err := b.CreateBucketIfNotExists([]byte(key.DeploymentID))
	if err != nil {
		return nil, errors.Wrapf(err, "creating sub-bucket for deploymentID %q", key.DeploymentID)
	}
	processSubBucket, err := deploymentSubBucket.CreateBucketIfNotExists(key.Serialize())
	if err != nil {
		return nil, errors.Wrapf(err, "creating sub-bucket for key: %v", key)
	}
	return processSubBucket, nil
}

func getProcessSubBucket(b *bbolt.Bucket, key *analystnotes.ProcessNoteKey) *bbolt.Bucket {
	deploymentSubBucket := b.Bucket([]byte(key.DeploymentID))
	if deploymentSubBucket == nil {
		return nil
	}
	return deploymentSubBucket.Bucket(key.Serialize())
}

func marshalAndPutComment(processSubBucket *bbolt.Bucket, key []byte, comment *storage.Comment) error {
	marshalled, err := proto.Marshal(comment)
	if err != nil {
		return errors.Wrap(err, "marshaling comment")
	}
	if err := processSubBucket.Put(key, marshalled); err != nil {
		return errors.Wrap(err, "inserting into process sub-bucket")
	}
	return nil
}

func (s *storeImpl) AddProcessComment(key *analystnotes.ProcessNoteKey, comment *storage.Comment) (string, error) {
	if err := validateKeyAndComment(key, comment); err != nil {
		return "", err
	}

	if comment.GetCommentId() != "" {
		return "", errors.Errorf("invalid comment: already has an id (%v)", comment)
	}

	var commentID string
	err := s.bucketRef.Update(func(b *bbolt.Bucket) error {
		processSubBucket, err := getOrCreateProcessSubBucket(b, key)
		if err != nil {
			return err
		}
		nextID, err := processSubBucket.NextSequence()
		if err != nil {
			return errors.Wrap(err, "getting next sequence id")
		}
		encodedKey := binenc.UVarInt(nextID)
		commentID = string(encodedKey)
		comment.CommentId = commentID
		now := types.TimestampNow()
		comment.CreatedAt = now
		addStandardFields(comment, now)

		if err := marshalAndPutComment(processSubBucket, encodedKey, comment); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return commentID, nil
}

func (s *storeImpl) UpdateProcessComment(key *analystnotes.ProcessNoteKey, comment *storage.Comment) error {
	if err := validateKeyAndComment(key, comment); err != nil {
		return err
	}

	if comment.GetCommentId() == "" {
		return errors.Errorf("invalid comment for update: no id specified (%v)", comment)
	}

	err := s.bucketRef.Update(func(b *bbolt.Bucket) error {
		processSubBucket, err := getOrCreateProcessSubBucket(b, key)
		if err != nil {
			return err
		}

		idBytes := []byte(comment.GetCommentId())
		existingCommentBytes := processSubBucket.Get(idBytes)
		if existingCommentBytes == nil {
			return errors.Errorf("no comment found for id: %v", comment.GetCommentId())
		}

		var existingComment storage.Comment
		if err := proto.Unmarshal(existingCommentBytes, &existingComment); err != nil {
			return errors.Errorf("unmarshaling existing comment: %v", err)
		}

		comment.CreatedAt = existingComment.GetCreatedAt()
		addStandardFields(comment, types.TimestampNow())

		if err := marshalAndPutComment(processSubBucket, idBytes, comment); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}
	return nil
}

func (s *storeImpl) GetComments(key *analystnotes.ProcessNoteKey) ([]*storage.Comment, error) {
	var comments []*storage.Comment
	err := s.bucketRef.View(func(b *bbolt.Bucket) error {
		processSubBucket := getProcessSubBucket(b, key)
		// No comments.
		if processSubBucket == nil {
			return nil
		}
		return processSubBucket.ForEach(func(_, v []byte) error {
			var comment storage.Comment
			if err := proto.Unmarshal(v, &comment); err != nil {
				return errors.Wrap(err, "proto unmarshaling")
			}
			comments = append(comments, &comment)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return comments, nil
}

func (s *storeImpl) GetCommentsCount(key *analystnotes.ProcessNoteKey) (int, error) {
	var count int
	err := s.bucketRef.View(func(b *bbolt.Bucket) error {
		processSubBucket := getProcessSubBucket(b, key)
		// No comments.
		if processSubBucket == nil {
			return nil
		}
		count = processSubBucket.Stats().KeyN
		return nil
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *storeImpl) GetComment(key *analystnotes.ProcessNoteKey, commentID string) (*storage.Comment, error) {
	var comment *storage.Comment
	err := s.bucketRef.View(func(b *bbolt.Bucket) error {
		processSubBucket := getProcessSubBucket(b, key)
		// No comments.
		if processSubBucket == nil {
			return nil
		}
		commentBytes := processSubBucket.Get([]byte(commentID))
		if commentBytes == nil {
			return nil
		}
		comment = new(storage.Comment)
		if err := proto.Unmarshal(commentBytes, comment); err != nil {
			return errors.Wrap(err, "proto unmarshaling")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return comment, nil
}

func (s *storeImpl) RemoveProcessComment(key *analystnotes.ProcessNoteKey, commentID string) error {
	if err := key.Validate(); err != nil {
		return err
	}
	return s.bucketRef.Update(func(b *bbolt.Bucket) error {
		processSubBucket := getProcessSubBucket(b, key)
		if processSubBucket == nil {
			return nil
		}
		return processSubBucket.Delete([]byte(commentID))
	})
}

func (s *storeImpl) RemoveAllProcessComments(key *analystnotes.ProcessNoteKey) error {
	if err := key.Validate(); err != nil {
		return err
	}

	return s.bucketRef.Update(func(b *bbolt.Bucket) error {
		deploymentSubBucket := b.Bucket([]byte(key.DeploymentID))
		if deploymentSubBucket == nil {
			return nil
		}
		return deploymentSubBucket.DeleteBucket(key.Serialize())
	})
}
