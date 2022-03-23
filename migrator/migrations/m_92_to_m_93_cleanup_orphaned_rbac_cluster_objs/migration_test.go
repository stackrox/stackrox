package m92tom93

import (
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	dbTypes "github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(cleanupAfterClusterTestSuite))
}

type cleanupAfterClusterTestSuite struct {
	suite.Suite

	db        *rocksdb.RocksDB
	databases *dbTypes.Databases
}

func (s *cleanupAfterClusterTestSuite) SetupTest() {
	rocksDB, err := rocksdb.NewTemp(s.T().Name())
	s.NoError(err)

	s.db = rocksDB
	s.databases = &dbTypes.Databases{RocksDB: rocksDB.DB}
}

func (s *cleanupAfterClusterTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(s.db)
}

func (s *cleanupAfterClusterTestSuite) TestMigrationRemovesOrphanedClusterObjects() {
	existingCluster := &storage.Cluster{
		Id:                 uuid.NewV4().String(),
		Name:               "Fake cluster 1",
		MainImage:          "docker.io/stackrox/rox:latest",
		CentralApiEndpoint: "central.stackrox:443",
	}

	key := rocksdbmigration.GetPrefixedKey(clusterBucket, []byte(existingCluster.GetId()))
	value, err := proto.Marshal(existingCluster)
	s.NoError(err)
	s.NoError(s.databases.RocksDB.Put(writeOpts, key, value))

	// Add in a cluster health status just to validate that it won't get picked up by the iterator
	chs := &storage.ClusterHealthStatus{SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY}
	chsKey := rocksdbmigration.GetPrefixedKey([]byte("clusters_health_status"), []byte(existingCluster.GetId()))
	chsValue, err := proto.Marshal(chs)
	s.NoError(err)
	s.NoError(s.databases.RocksDB.Put(writeOpts, chsKey, chsValue))

	validSAs, toRemoveSAs := s.getServiceAccounts(existingCluster)
	validRoles, toRemoveRoles := s.getK8SRoles(existingCluster)
	validBindings, toRemoveBindings := s.getRoleBindings(existingCluster)

	err = cleanupOrphanedRBACObjectsFromDeletedClusters(s.databases)
	s.NoError(err)

	s.validateObjectsRemoved(serviceAccountBucket, validSAs, toRemoveSAs, func(data []byte) string {
		sa := &storage.ServiceAccount{}
		s.NoError(proto.Unmarshal(data, sa))
		return sa.GetId()
	})
	s.validateObjectsRemoved(k8sRoleBucket, validRoles, toRemoveRoles, func(data []byte) string {
		sa := &storage.ServiceAccount{}
		s.NoError(proto.Unmarshal(data, sa))
		return sa.GetId()
	})
	s.validateObjectsRemoved(roleBindingsBucket, validBindings, toRemoveBindings, func(data []byte) string {
		sa := &storage.ServiceAccount{}
		s.NoError(proto.Unmarshal(data, sa))
		return sa.GetId()
	})
}

func (s *cleanupAfterClusterTestSuite) validateObjectsRemoved(bucket []byte, validIDs set.StringSet, idsToRemove set.StringSet, getID func(data []byte) string) {
	it := s.databases.RocksDB.NewIterator(readOpts)
	defer it.Close()

	var foundObjects int
	for it.Seek(bucket); it.ValidForPrefix(bucket); it.Next() {
		id := getID(it.Value().Data())
		s.True(validIDs.Contains(id))
		s.False(idsToRemove.Contains(id))
		foundObjects++
	}
	s.Equal(validIDs.Cardinality(), foundObjects)
}

func (s *cleanupAfterClusterTestSuite) getServiceAccounts(existingCluster *storage.Cluster) (set.StringSet, set.StringSet) {
	var validSAs set.StringSet
	var toRemoveSAs set.StringSet
	for i := 0; i < 20; i++ {
		clusterID := existingCluster.GetId()
		id := uuid.NewV4().String()
		if i%2 == 0 {
			validSAs.Add(id)
		} else {
			toRemoveSAs.Add(id)
			clusterID = uuid.NewV4().String()
		}
		sa := &storage.ServiceAccount{
			Id:        id,
			Name:      fmt.Sprintf("Fake SA %d", i),
			ClusterId: clusterID,
			// Add field that is only for ServiceAccount, so that proto unmarshalling will fail if you try to unmarshal to something else
			Secrets: []string{"blah"},
		}

		key := rocksdbmigration.GetPrefixedKey(serviceAccountBucket, []byte(sa.GetId()))
		value, err := proto.Marshal(sa)
		s.NoError(err)
		s.NoError(s.databases.RocksDB.Put(writeOpts, key, value))
	}
	return validSAs, toRemoveSAs
}

func (s *cleanupAfterClusterTestSuite) getK8SRoles(existingCluster *storage.Cluster) (set.StringSet, set.StringSet) {
	var validRoles set.StringSet
	var toRemoveRoles set.StringSet
	for i := 0; i < 20; i++ {
		clusterID := existingCluster.GetId()
		id := uuid.NewV4().String()
		if i%2 == 0 {
			validRoles.Add(id)
		} else {
			toRemoveRoles.Add(id)
			clusterID = uuid.NewV4().String()
		}
		role := &storage.K8SRole{
			Id:        id,
			Name:      fmt.Sprintf("Fake Role %d", i),
			ClusterId: clusterID,
			// Add field that is only for K8SRole, so that proto unmarshalling will fail if you try to unmarshal to something else
			Rules: []*storage.PolicyRule{{Verbs: []string{"FAKE-VERB"}}},
		}

		key := rocksdbmigration.GetPrefixedKey(k8sRoleBucket, []byte(role.GetId()))
		value, err := proto.Marshal(role)
		s.NoError(err)
		s.NoError(s.databases.RocksDB.Put(writeOpts, key, value))
	}
	return validRoles, toRemoveRoles
}

func (s *cleanupAfterClusterTestSuite) getRoleBindings(existingCluster *storage.Cluster) (set.StringSet, set.StringSet) {
	var validBindings set.StringSet
	var toRemoveBindings set.StringSet
	for i := 0; i < 20; i++ {
		clusterID := existingCluster.GetId()
		id := uuid.NewV4().String()
		if i%2 == 0 {
			validBindings.Add(id)
		} else {
			toRemoveBindings.Add(id)
			clusterID = uuid.NewV4().String()
		}
		rb := &storage.K8SRoleBinding{
			Id:        id,
			Name:      fmt.Sprintf("Fake binding %d", i),
			ClusterId: clusterID,
			// Add field that is only for RoleBinding, so that proto unmarshalling will fail if you try to unmarshal to something else
			Subjects: []*storage.Subject{{Id: uuid.NewV4().String(), Kind: storage.SubjectKind_SERVICE_ACCOUNT}},
		}

		key := rocksdbmigration.GetPrefixedKey(roleBindingsBucket, []byte(rb.GetId()))
		value, err := proto.Marshal(rb)
		s.NoError(err)
		s.NoError(s.databases.RocksDB.Put(writeOpts, key, value))
	}
	return validBindings, toRemoveBindings
}
