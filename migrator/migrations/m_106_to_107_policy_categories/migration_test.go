package m106to107

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	dbTypes "github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
	"github.com/tecbot/gorocksdb"
	bolt "go.etcd.io/bbolt"
)

var (
	userDefinedCategory1 = "New Category 1"
	userDefinedCategory2 = "New Category 2"

	policy = &storage.Policy{
		Id:              "b3523d84-ac1a-4daa-a908-62d196c5a741",
		Name:            "Vulnerable Container",
		Categories:      []string{userDefinedCategory1, userDefinedCategory2},
		Description:     "Alert if the container contains vulnerabilities",
		Severity:        storage.Severity_LOW_SEVERITY,
		Rationale:       "This is the rationale",
		Remediation:     "This is the remediation",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_BUILD},
		Scope: []*storage.Scope{
			{
				Cluster:   "prod cluster",
				Namespace: "stackrox",
				Label: &storage.Scope_Label{
					Key:   "com.docker.stack.namespace",
					Value: "prevent",
				},
			},
		},
		PolicyVersion: "1.1",
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Image Registry",
						Values: []*storage.PolicyValue{
							{
								Value: "docker.io",
							},
						},
					},
					{
						FieldName: "CVE",
						Values: []*storage.PolicyValue{
							{
								Value: "CVE-1234",
							},
						},
					},
				},
			},
		},
	}
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(categoriesRocksDBMigrationTestSuite))
}

type categoriesRocksDBMigrationTestSuite struct {
	suite.Suite

	db        *rocksdb.RocksDB
	databases *dbTypes.Databases
}

func (suite *categoriesRocksDBMigrationTestSuite) SetupTest() {
	boltdb := testutils.DBForT(suite.T())
	suite.NoError(boltdb.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(policiesBucket); err != nil {
			return err
		}
		return nil
	}))

	rocksDB, err := rocksdb.NewTemp(suite.T().Name())
	suite.NoError(err)

	suite.db = rocksDB
	suite.databases = &dbTypes.Databases{BoltDB: boltdb, RocksDB: rocksDB.DB}
}

func (suite *categoriesRocksDBMigrationTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.databases.BoltDB)
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *categoriesRocksDBMigrationTestSuite) TestCategoriesMigrationToRocksDB() {
	boltDB := suite.databases.BoltDB
	rocksDB := suite.databases.RocksDB

	// Insert policy
	suite.NoError(boltDB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policiesBucket)
		bytes, err := proto.Marshal(policy)
		if err != nil {
			return err
		}
		if err := bucket.Put([]byte(policy.GetName()), bytes); err != nil {
			return err
		}
		return nil
	}))

	// Run migration.
	suite.NoError(addUserDefinedCategories(boltDB, rocksDB))

	categoriesAfterMigration := make([]string, 0, 2)

	it := rocksDB.NewIterator(gorocksdb.NewDefaultReadOptions())
	defer it.Close()
	for it.Seek(categoriesBucket); it.ValidForPrefix(categoriesBucket); it.Next() {
		var c storage.PolicyCategory
		if err := proto.Unmarshal(it.Value().Data(), &c); err != nil {
			suite.NoError(err)
		}
		categoriesAfterMigration = append(categoriesAfterMigration, c.Name)
	}
	suite.ElementsMatch(categoriesAfterMigration, []string{userDefinedCategory1, userDefinedCategory2})
}
