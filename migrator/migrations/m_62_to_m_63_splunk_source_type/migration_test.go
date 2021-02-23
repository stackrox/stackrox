package m62tom63

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestMigrateSplunkSourceType(t *testing.T) {
	suite.Run(t, new(migrateSplunkSourceTypeTestSuite))
}

type migrateSplunkSourceTypeTestSuite struct {
	suite.Suite

	db *bolt.DB
}

func (suite *migrateSplunkSourceTypeTestSuite) SetupTest() {
	db, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.NoError(db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(notifiersBucket)
		return err
	}))
	suite.db = db
}

func (suite *migrateSplunkSourceTypeTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func (suite *migrateSplunkSourceTypeTestSuite) TestMigration() {
	noDerivedField := &storage.Notifier{
		Id:   "1",
		Name: "no-derived",
		Type: "splunk",
		Config: &storage.Notifier_Splunk{
			Splunk: &storage.Splunk{
				DerivedSourceTypeDeprecated: &storage.Splunk_DerivedSourceType{
					DerivedSourceType: false,
				},
			},
		},
	}
	derivedField := &storage.Notifier{
		Id:   "2",
		Name: "derived",
		Type: "splunk",
		Config: &storage.Notifier_Splunk{
			Splunk: &storage.Splunk{
				DerivedSourceTypeDeprecated: &storage.Splunk_DerivedSourceType{
					DerivedSourceType: true,
				},
			},
		},
	}
	err := suite.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(notifiersBucket)

		data, err := proto.Marshal(noDerivedField)
		suite.NoError(err)
		suite.NoError(bucket.Put([]byte(noDerivedField.GetId()), data))

		data, err = proto.Marshal(derivedField)
		suite.NoError(err)
		suite.NoError(bucket.Put([]byte(derivedField.GetId()), data))
		return nil
	})
	suite.NoError(err)

	suite.NoError(migrateSplunkSourceType(suite.db))

	expectedNoDerivedField := noDerivedField.Clone()
	expectedNoDerivedField.GetSplunk().SourceTypes = map[string]string{
		"alert": jsonSourceType,
		"audit": jsonSourceType,
	}
	expectedNoDerivedField.GetSplunk().DerivedSourceTypeDeprecated = nil

	expectedDerivedField := derivedField.Clone()
	expectedDerivedField.GetSplunk().SourceTypes = map[string]string{
		"alert": "stackrox-alert",
		"audit": "stackrox-audit-message",
	}
	expectedDerivedField.GetSplunk().DerivedSourceTypeDeprecated = nil
	err = suite.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(notifiersBucket)

		notifier := &storage.Notifier{}
		suite.NoError(proto.Unmarshal(bucket.Get([]byte(noDerivedField.GetId())), notifier))
		suite.Equal(expectedNoDerivedField, notifier)

		notifier = &storage.Notifier{}
		suite.NoError(proto.Unmarshal(bucket.Get([]byte(derivedField.GetId())), notifier))
		suite.Equal(expectedDerivedField, notifier)

		return nil
	})
	suite.NoError(err)
}
