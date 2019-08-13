package m17tom18

import (
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/golang/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

type migrationTestSuite struct {
	suite.Suite

	db *bolt.DB
}

func (suite *migrationTestSuite) SetupTest() {
	db, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.NoError(db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(configBucket)
		return err
	}))
	suite.db = db
}

func (suite *migrationTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func insertThing(bucket bolthelpers.BucketRef, id string, pb proto.Message) error {
	return bucket.Update(func(b *bolt.Bucket) error {
		bytes, err := proto.Marshal(pb)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), bytes)
	})
}

func (suite *migrationTestSuite) TestConfigMigration() {
	ref, err := bolthelpers.TopLevelRefWithCreateIfNotExists(suite.db, configBucket)
	suite.NoError(err)
	config := &storage.Config{
		PublicConfig: &storage.PublicConfig{
			LoginNotice: &storage.LoginNotice{
				Enabled: true,
				Text:    "Login Notice",
			},
		},
		PrivateConfig: &storage.PrivateConfig{
			AlertRetention: &storage.PrivateConfig_DEPRECATEDAlertRetentionDurationDays{
				DEPRECATEDAlertRetentionDurationDays: 30,
			},
			ImageRetentionDurationDays: 27,
		},
	}
	suite.NoError(insertThing(ref, "\x00", config))
	suite.NoError(migration.Run(suite.db, nil))
	suite.NoError(ref.View(func(b *bolt.Bucket) error {
		return b.ForEach(func(k, v []byte) error {
			suite.Equal(k, []byte("\x00"))
			config := &storage.Config{}
			suite.NoError(proto.Unmarshal(v, config))
			suite.Equal("Login Notice", config.GetPublicConfig().GetLoginNotice().GetText())
			suite.Equal(true, config.GetPublicConfig().GetLoginNotice().GetEnabled())
			suite.Equal(int32(27), config.GetPrivateConfig().GetImageRetentionDurationDays())
			suite.IsType(new(storage.PrivateConfig_AlertConfig), config.GetPrivateConfig().GetAlertRetention())
			suite.Equal(int32(0), config.GetPrivateConfig().GetAlertConfig().GetAllRuntimeRetentionDurationDays())
			suite.Equal(int32(30), config.GetPrivateConfig().GetAlertConfig().GetDeletedRuntimeRetentionDurationDays())
			suite.Equal(int32(30), config.GetPrivateConfig().GetAlertConfig().GetResolvedDeployRetentionDurationDays())
			return nil
		})
	}))
}
