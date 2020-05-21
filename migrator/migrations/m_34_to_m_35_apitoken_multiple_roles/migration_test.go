package m33tom34

import (
	"testing"

	"github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	unmigratedTokens = []*storage.TokenMetadata{
		{
			Id:         "0",
			Name:       "token0",
			Role:       "foo",
			IssuedAt:   types.TimestampNow(),
			Expiration: types.TimestampNow(),
		},
		{
			Id:         "1",
			Name:       "token1",
			Role:       "bar",
			IssuedAt:   types.TimestampNow(),
			Expiration: types.TimestampNow(),
		},
	}

	unmigratedTokensAfterMigration = []*storage.TokenMetadata{
		{
			Id:         "0",
			Name:       "token0",
			Role:       "foo",
			Roles:      []string{"foo"},
			IssuedAt:   unmigratedTokens[0].IssuedAt,
			Expiration: unmigratedTokens[0].Expiration,
		},
		{
			Id:         "1",
			Name:       "token1",
			Role:       "bar",
			Roles:      []string{"bar"},
			IssuedAt:   unmigratedTokens[1].IssuedAt,
			Expiration: unmigratedTokens[1].Expiration,
		},
	}

	alreadyMigratedTokens = []*storage.TokenMetadata{
		{
			Id:         "2",
			Name:       "token2",
			Role:       "baz",
			Roles:      []string{"baz"},
			IssuedAt:   types.TimestampNow(),
			Expiration: types.TimestampNow(),
		},
		{
			Id:         "3",
			Name:       "token3",
			Roles:      []string{"qux", "quux"},
			IssuedAt:   types.TimestampNow(),
			Expiration: types.TimestampNow(),
		},
	}
)

func TestAPITokenRoleMigration(t *testing.T) {
	db := testutils.DBForT(t)

	var tokensToUpsert []*storage.TokenMetadata
	tokensToUpsert = append(tokensToUpsert, unmigratedTokens...)
	tokensToUpsert = append(tokensToUpsert, alreadyMigratedTokens...)

	require.NoError(t, db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucket(apiTokensBucket)
		if err != nil {
			return err
		}

		for _, token := range tokensToUpsert {
			bytes, err := proto.Marshal(token)
			if err != nil {
				return err
			}
			if err := bucket.Put([]byte(token.GetId()), bytes); err != nil {
				return err
			}
		}
		return nil
	}))

	require.NoError(t, migrateAPITokenInfo(db))

	var allTokensAfterMigration []*storage.TokenMetadata

	require.NoError(t, db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(apiTokensBucket)
		if bucket == nil {
			return errors.New("bucket does not exist")
		}
		return bucket.ForEach(func(k, v []byte) error {
			tokenMD := &storage.TokenMetadata{}
			if err := proto.Unmarshal(v, tokenMD); err != nil {
				return err
			}
			if string(k) != tokenMD.GetId() {
				return errors.Errorf("ID mismatch: %s vs %s", k, tokenMD.GetId())
			}
			allTokensAfterMigration = append(allTokensAfterMigration, tokenMD)
			return nil
		})
	}))

	var expectedTokensAfterMigration []*storage.TokenMetadata
	expectedTokensAfterMigration = append(expectedTokensAfterMigration, unmigratedTokensAfterMigration...)
	expectedTokensAfterMigration = append(expectedTokensAfterMigration, alreadyMigratedTokens...)

	assert.ElementsMatch(t, expectedTokensAfterMigration, allTokensAfterMigration)
}
