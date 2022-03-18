package m95tom96

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/rockshelper"
	dbTypes "github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
	"github.com/tecbot/gorocksdb"
)

var (
	alertBucket = []byte("alerts")
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(alertScopeInfoCopyTestSuite))
}

type alertScopeInfoCopyTestSuite struct {
	suite.Suite

	db        *rocksdb.RocksDB
	databases *dbTypes.Databases
}

func (s *alertScopeInfoCopyTestSuite) SetupTest() {
	rocksDB, err := rocksdb.NewTemp(s.T().Name())
	s.NoError(err)

	s.db = rocksDB
	s.databases = &dbTypes.Databases{RocksDB: rocksDB.DB}
}

func (s *alertScopeInfoCopyTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(s.db)
}

func (s *alertScopeInfoCopyTestSuite) writeAlertToStore(alert *storage.Alert) {
	writeOpts := gorocksdb.NewDefaultWriteOptions()
	value, err := proto.Marshall(alert)
	s.NoError(err)
	err = s.db.Put(writeOpts,
		rocksdbmigration.GetPrefixedKey(alertBucket, []byte(alert.GetId())),
		value)
	s.NoError(err)
}

func (s *alertScopeInfoCopyTestSuite) checkMigratedAlertScopingInformation(alertID, clusterID, clusterName, namespace, namespaceID string) {
	readOpts := gorocksdb.NewDefaultReadOptions()
	msg, exists, err := rockshelper.ReadFromRocksDB(s.db, readOpts, &storage.Alert{}, alertBucket, []byte(alertID))
	s.NoError(err)
	s.True(exists)
	readAlert := msg.(*storage.Alert)
	s.Equal(clusterID, readAlert.ClusterId)
	s.Equal(clusterName, readAlert.ClusterName)
	s.Equal(namespace, readAlert.Namespace)
	s.Equal(namespaceID, readAlert.NamespaceId)
}

func (s *alertScopeInfoCopyTestSuite) TestDeploymentAlertScopingInformationCopy() {
	alert := fixtures.GetAlert()
	entity := alert.GetDeployment()
	s.writeAlertToStore(alert)

	err := copyAlertScopingInformationToRoot(s.db)
	s.NoError(err)

	s.checkMigratedAlertScopingInformation(alert.GetId(), entity.GetClusterId(), entity.GetClusterName(), entity.GetNamespace(), entity.GetNamespaceId())
}

func (s *alertScopeInfoCopyTestSuite) TestResourceAlertScopingInformationCopy() {
	alert := fixtures.GetResourceAlert()
	entity := alert.GetResource()
	s.writeAlertToStore(alert)

	err := copyAlertScopingInformationToRoot(s.db)
	s.NoError(err)

	s.checkMigratedAlertScopingInformation(alert.GetId(), entity.GetClusterId(), entity.GetClusterName(), entity.GetNamespace(), entity.GetNamespaceId())
}

func (s *alertScopeInfoCopyTestSuite) TestImageAlertScopingInformationCopy() {
	alert := fixtures.GetImageAlert()
	s.writeAlertToStore(alert)

	err := copyAlertScopingInformationToRoot(s.db)
	s.NoError(err)

	s.checkMigratedAlertScopingInformation(alert.GetId(), "", "", "", "")
}