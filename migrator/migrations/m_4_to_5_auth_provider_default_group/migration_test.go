package m4to5

import (
	"fmt"
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

const (
	arbitraryRoleName = "ARBITRARY_ROLE"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(MigrationTestSuite))
}

type MigrationTestSuite struct {
	suite.Suite

	db *bolt.DB
}

func (suite *MigrationTestSuite) SetupTest() {
	db, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.NoError(db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(authProvidersBucketName); err != nil {
			return err
		}
		_, err := tx.CreateBucketIfNotExists(groupsBucketName)
		return err
	}))
	suite.db = db
}

func (suite *MigrationTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func getAuthProvider(id string, basic bool) *storage.AuthProvider {
	typ := "arbitrary_type"
	if basic {
		typ = basicAuthProviderTypeName
	}
	return &storage.AuthProvider{
		Id:   id,
		Type: typ,
	}
}

func (suite *MigrationTestSuite) mustInsertAuthProvider(id string, basic bool) {
	authProvidersBucket := bolthelpers.TopLevelRef(suite.db, authProvidersBucketName)
	suite.NoError(authProvidersBucket.Update(func(b *bolt.Bucket) error {
		bytes, err := proto.Marshal(getAuthProvider(id, basic))
		if err != nil {
			return err
		}
		return b.Put([]byte(id), bytes)
	}))
}

func (suite *MigrationTestSuite) mustInsertGroupForAuthProvider(id, key, value string) {
	groupsBucket := bolthelpers.TopLevelRef(suite.db, groupsBucketName)
	suite.NoError(groupsBucket.Update(func(b *bolt.Bucket) error {
		return b.Put([]byte(fmt.Sprintf("%s:%s:%s", id, key, value)), []byte(arbitraryRoleName))
	}))
}

func (suite *MigrationTestSuite) TestAuthProviderMigration() {
	goodIds := []string{"id1", "id2", "id3", "id4"}
	goodIDKeys := []string{"", "key1", "", "key2"}
	goodIDValues := []string{"", "", "value1", "value2"}
	badIds := []string{"badid1", "badid2"}

	for i, id := range goodIds {
		suite.mustInsertAuthProvider(id, false)
		suite.mustInsertGroupForAuthProvider(id, goodIDKeys[i], goodIDValues[i])
	}
	for _, id := range badIds {
		suite.mustInsertAuthProvider(id, false)
	}
	suite.mustInsertAuthProvider("BASIC", true)

	suite.NoError(migration.Run(suite.db, nil))

	type keyValuePair struct {
		key   string
		value string
	}
	var keyValuePairs []keyValuePair

	groupsBucket := bolthelpers.TopLevelRef(suite.db, groupsBucketName)
	suite.NoError(groupsBucket.View(func(b *bolt.Bucket) error {
		return b.ForEach(func(k, v []byte) error {
			keyValuePairs = append(keyValuePairs, keyValuePair{string(k), string(v)})
			return nil
		})
	}))

	var expectedKeyValuePairs []keyValuePair
	for i, id := range goodIds {
		expectedKeyValuePairs = append(expectedKeyValuePairs, keyValuePair{fmt.Sprintf("%s:%s:%s", id, goodIDKeys[i], goodIDValues[i]), arbitraryRoleName})
	}
	for _, id := range badIds {
		expectedKeyValuePairs = append(expectedKeyValuePairs, keyValuePair{fmt.Sprintf("%s::", id), "Admin"})
	}
	suite.ElementsMatch(expectedKeyValuePairs, keyValuePairs)
}
