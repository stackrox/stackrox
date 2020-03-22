package m14tom15

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
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
		_, err := tx.CreateBucketIfNotExists(rolesBucketName)
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

func (suite *migrationTestSuite) mustInsertRoles(role *storage.Role) {
	rolesBucket := bolthelpers.TopLevelRef(suite.db, rolesBucketName)
	suite.NoError(insertThing(rolesBucket, role.GetName(), role))
}

func (suite *migrationTestSuite) TestPurgeDebugMetricsMigration() {
	oldRoles := []*storage.Role{
		{
			Name: "Role1",
			ResourceToAccess: map[string]storage.Access{
				"DebugMetrics":  storage.Access_READ_WRITE_ACCESS,
				"otherResource": storage.Access_READ_ACCESS,
			},
		},
		{
			Name: "Role2",
			ResourceToAccess: map[string]storage.Access{
				"otherResource": storage.Access_READ_ACCESS,
			},
		},
		{
			Name: "Role3",
			ResourceToAccess: map[string]storage.Access{
				"DebugMetrics": storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			Name: "Role4",
		},
	}

	expectedRoles := []*storage.Role{
		{
			Name: "Role1",
			ResourceToAccess: map[string]storage.Access{
				"otherResource": storage.Access_READ_ACCESS,
			},
		},
		{
			Name: "Role2",
			ResourceToAccess: map[string]storage.Access{
				"otherResource": storage.Access_READ_ACCESS,
			},
		},
		{
			Name: "Role3",
		},
		{
			Name: "Role4",
		},
	}

	for _, role := range oldRoles {
		suite.mustInsertRoles(role)
	}

	suite.NoError(migration.Run(suite.db, nil))

	newRoles := make([]*storage.Role, 0, len(oldRoles))
	rolesBucket := bolthelpers.TopLevelRef(suite.db, rolesBucketName)
	suite.NoError(rolesBucket.View(func(b *bolt.Bucket) error {
		return b.ForEach(func(_, v []byte) error {
			var role storage.Role
			err := proto.Unmarshal(v, &role)
			if err != nil {
				return err
			}
			newRoles = append(newRoles, &role)
			return nil
		})
	}))
	suite.ElementsMatch(expectedRoles, newRoles)
}
