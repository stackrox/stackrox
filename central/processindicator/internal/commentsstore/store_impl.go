package commentsstore

import (
	"fmt"

	"github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/comments"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/binenc"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	processCommentKeyNS = uuid.FromStringOrPanic("c19c0cea-b5df-40c4-80e7-836a1b0785e6")
)

func serializeProcessKey(key *comments.ProcessCommentKey) string {
	return uuid.NewV5(processCommentKeyNS, fmt.Sprintf("%s\x00%s\x00:%s", key.ContainerName, key.ExecFilePath, key.Args)).String()
}

type storeImpl struct {
	bucketRef bolthelper.BucketRef
}

func addStandardFields(comment *storage.Comment, lastModified *types.Timestamp) {
	comment.ResourceType = storage.ResourceType_PROCESS
	comment.LastModified = lastModified
}

func validateKeyAndComment(key *comments.ProcessCommentKey, comment *storage.Comment) error {
	if err := key.Validate(); err != nil {
		return err
	}

	if comment == nil {
		return errors.New("invalid comment: was nil")
	}

	return nil
}

func getOrCreateProcessSubBucket(b *bbolt.Bucket, key *comments.ProcessCommentKey) (*bbolt.Bucket, error) {
	deploymentSubBucket, err := b.CreateBucketIfNotExists([]byte(key.DeploymentID))
	if err != nil {
		return nil, errors.Wrapf(err, "creating sub-bucket for deploymentID %q", key.DeploymentID)
	}
	processSubBucket, err := deploymentSubBucket.CreateBucketIfNotExists([]byte(serializeProcessKey(key)))
	if err != nil {
		return nil, errors.Wrapf(err, "creating sub-bucket for key: %v", key)
	}
	return processSubBucket, nil
}

func getProcessSubBucket(b *bbolt.Bucket, key *comments.ProcessCommentKey) *bbolt.Bucket {
	deploymentSubBucket := b.Bucket([]byte(key.DeploymentID))
	if deploymentSubBucket == nil {
		return nil
	}
	return deploymentSubBucket.Bucket([]byte(serializeProcessKey(key)))
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

func (s *storeImpl) AddProcessComment(key *comments.ProcessCommentKey, comment *storage.Comment) (string, error) {
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

func (s *storeImpl) UpdateProcessComment(key *comments.ProcessCommentKey, comment *storage.Comment) error {
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

func (s *storeImpl) GetCommentsForProcessKey(key *comments.ProcessCommentKey) ([]*storage.Comment, error) {
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

func (s *storeImpl) GetComment(key *comments.ProcessCommentKey, commentID string) (*storage.Comment, error) {
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

func (s *storeImpl) RemoveProcessComment(key *comments.ProcessCommentKey, commentID string) error {
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

func (s *storeImpl) RemoveAllProcessComments(key *comments.ProcessCommentKey) error {
	if err := key.Validate(); err != nil {
		return err
	}

	return s.bucketRef.Update(func(b *bbolt.Bucket) error {
		deploymentSubBucket := b.Bucket([]byte(key.DeploymentID))
		if deploymentSubBucket == nil {
			return nil
		}
		return deploymentSubBucket.DeleteBucket([]byte(serializeProcessKey(key)))
	})
}
