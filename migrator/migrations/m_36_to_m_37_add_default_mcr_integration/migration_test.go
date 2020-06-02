package m36tom37

import (
	"testing"

	"github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
)

var (
	imageIntegrationsBucket = []byte("imageintegrations")
)

func TestDefaultMCRIntegrationMigration(t *testing.T) {
	db := testutils.DBForT(t)

	require.NoError(t, db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucket(imageIntegrationBucket)
		if err != nil {
			return err
		}
		return nil
	}))

	require.NoError(t, addDefaultMCRIntegration(db))

	var imageIntegrations []*storage.ImageIntegration

	require.NoError(t, db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(imageIntegrationsBucket)
		if bucket == nil {
			return errors.New("bucket does not exist")
		}
		return bucket.ForEach(func(k, v []byte) error {
			integration := &storage.ImageIntegration{}
			if err := proto.Unmarshal(v, integration); err != nil {
				return err
			}
			if string(k) != integration.GetId() {
				return errors.Errorf("ID mismatch: %s vs %s", k, integration.GetId())
			}
			imageIntegrations = append(imageIntegrations, integration)
			return nil
		})
	}))

	assert.Equal(t, len(imageIntegrations), 1)
}
