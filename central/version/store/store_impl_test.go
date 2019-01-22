package store

import (
	"fmt"
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stretchr/testify/suite"
)

func TestVersionStore(t *testing.T) {
	suite.Run(t, new(VersionStoreTestSuite))
}

type VersionStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store Store
}

func (suite *VersionStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store = New(db)
}

func (suite *VersionStoreTestSuite) TearDownSuite() {
	suite.NoError(suite.db.Close())
}

func (suite *VersionStoreTestSuite) TestVersionStore() {
	v, err := suite.store.GetVersion()
	suite.NoError(err)
	suite.Nil(v)

	for _, version := range []int32{2, 5, 19} {
		protoVersion := &storage.Version{SeqNum: version, Version: fmt.Sprintf("Version %d", version)}
		suite.NoError(suite.store.UpdateVersion(protoVersion))
		got, err := suite.store.GetVersion()
		suite.NoError(err)
		suite.Equal(protoVersion, got)
	}
}
