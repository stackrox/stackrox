package store

import (
	"errors"
	"fmt"
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	ops "github.com/stackrox/rox/pkg/metrics"
)

type storeImpl struct {
	*bolt.DB
}

func (b *storeImpl) AddToken(token *storage.TokenMetadata) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "APIToken")

	if token.GetId() == "" {
		// This is most likely a programming error.
		return errors.New("token ID is empty")
	}

	bytes, err := proto.Marshal(token)
	if err != nil {
		return fmt.Errorf("proto marshaling: %s", err)
	}

	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(apiTokensBucket)
		return bucket.Put([]byte(token.GetId()), bytes)
	})
}

func (b *storeImpl) GetTokenOrNil(id string) (token *storage.TokenMetadata, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "APIToken")

	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(apiTokensBucket)
		tokenBytes := bucket.Get([]byte(id))
		if tokenBytes == nil {
			return nil
		}
		token = new(storage.TokenMetadata)
		err := proto.Unmarshal(tokenBytes, token)
		if err != nil {
			return fmt.Errorf("proto unmarshaling: %s", err)
		}
		return nil
	})
	return
}

func (b *storeImpl) GetTokens(req *v1.GetAPITokensRequest) (tokens []*storage.TokenMetadata, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetAll, "APIToken")

	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(apiTokensBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var token storage.TokenMetadata
			err := proto.Unmarshal(v, &token)
			if err != nil {
				return fmt.Errorf("proto unmarshaling: %s", err)
			}
			// If the request specifies a value for revoked, make sure the value matches.
			if req.GetRevokedOneof() != nil && req.GetRevoked() != token.GetRevoked() {
				return nil
			}
			tokens = append(tokens, &token)
			return nil
		})
	})

	return
}

func (b *storeImpl) RevokeToken(id string) (exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "RevokedTokenID")

	token, err := b.GetTokenOrNil(id)
	if token == nil {
		return
	}
	exists = true
	token.Revoked = true
	bytes, err := proto.Marshal(token)
	if err != nil {
		return
	}

	err = b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(apiTokensBucket)
		return bucket.Put([]byte(id), bytes)
	})
	return
}
