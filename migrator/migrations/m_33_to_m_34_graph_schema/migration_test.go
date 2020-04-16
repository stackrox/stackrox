package m33tom34

import (
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/migrator/badgerhelpers"
	"github.com/stretchr/testify/suite"
)

var (
	clusterID1 = "c1"
	clusterID2 = "c2"

	namespaceID1 = "nid1"
	namespaceID2 = "nid2"
	namespaceID3 = "nid3"
	namespaceID4 = "nid4"
	namespaceID5 = "nid5"

	namespaceName1 = "nn1"
	namespaceName2 = "nn2"
	namespaceName3 = "nn3"

	deploymentID1  = "did1"
	deploymentID2  = "did2"
	deploymentID3  = "did3"
	deploymentID4  = "did4"
	deploymentID5  = "did5"
	deploymentID6  = "did6"
	deploymentID7  = "did7"
	deploymentID8  = "did8"
	deploymentID9  = "did9"
	deploymentID10 = "did10"
)

func TestDackBoxMigration(t *testing.T) {
	suite.Run(t, new(DackBoxMigrationTestSuite))
}

type DackBoxMigrationTestSuite struct {
	suite.Suite

	db *badger.DB
}

func (suite *DackBoxMigrationTestSuite) SetupSuite() {
	var err error
	suite.db, err = badgerhelpers.NewTemp("graph_schema_migration_test")
	if err != nil {
		suite.FailNowf("failed to create DB: %+v", err.Error())
	}
}

func (suite *DackBoxMigrationTestSuite) TearDownSuite() {
	_ = suite.db.Close()
}

func (suite *DackBoxMigrationTestSuite) TestGraphMigrationMixed() {
	// Set up the mappings as they would currently appear.
	// Clusters point to both their namespace IDs and their namespace names.
	clusterMappings := map[string]SortedKeys{
		string(getClusterKey([]byte(clusterID1))): [][]byte{
			getNamespaceKey([]byte(namespaceID1)),
			getNamespaceKey([]byte(namespaceID2)),
			getNamespaceKey([]byte(namespaceID3)),
			getNamespaceSACKey([]byte(namespaceName1)),
			getNamespaceSACKey([]byte(namespaceName2)),
			getNamespaceSACKey([]byte(namespaceName3)),
		},
		string(getClusterKey([]byte(clusterID2))): [][]byte{
			getNamespaceKey([]byte(namespaceID4)),
			getNamespaceKey([]byte(namespaceID5)),
			getNamespaceSACKey([]byte(namespaceName1)), // Namespace ID 1 and 4 have the same Namespace Name
			getNamespaceSACKey([]byte(namespaceName3)), // Namespace ID 3 and 5 have the same Namespace Name
		},
	}

	// Namespace IDs point to their deployments.
	namespaceMappings := map[string]SortedKeys{
		string(getNamespaceKey([]byte(namespaceID1))): [][]byte{
			getDeploymentKey([]byte(deploymentID1)),
			getDeploymentKey([]byte(deploymentID2)),
		},
		string(getNamespaceKey([]byte(namespaceID2))): [][]byte{
			getDeploymentKey([]byte(deploymentID3)),
			getDeploymentKey([]byte(deploymentID4)),
		},
		string(getNamespaceKey([]byte(namespaceID3))): [][]byte{
			getDeploymentKey([]byte(deploymentID5)),
			getDeploymentKey([]byte(deploymentID6)),
			getDeploymentKey([]byte(deploymentID7)),
			getDeploymentKey([]byte(deploymentID8)),
		},
		string(getNamespaceKey([]byte(namespaceID4))): [][]byte{
			getDeploymentKey([]byte(deploymentID9)),
			getDeploymentKey([]byte(deploymentID10)),
		},
	}

	// Namespace Names point to their deployments.
	namespaceSACMappings := map[string]SortedKeys{
		string(getNamespaceSACKey([]byte(namespaceName1))): [][]byte{
			getDeploymentKey([]byte(deploymentID1)),
			getDeploymentKey([]byte(deploymentID2)),
			getDeploymentKey([]byte(deploymentID9)), // Namespace ID 1 and 4 have the same Namespace Name
			getDeploymentKey([]byte(deploymentID10)),
		},
		string(getNamespaceSACKey([]byte(namespaceName2))): [][]byte{
			getDeploymentKey([]byte(deploymentID3)),
			getDeploymentKey([]byte(deploymentID4)),
		},
		string(getNamespaceSACKey([]byte(namespaceName3))): [][]byte{
			getDeploymentKey([]byte(deploymentID5)),
			getDeploymentKey([]byte(deploymentID6)),
			getDeploymentKey([]byte(deploymentID7)),
			getDeploymentKey([]byte(deploymentID8)),
			// Namespace ID 3 and 5 have the same Namespace Name, but 5 has no deployments.
		},
	}

	// Write old versions of the deployments and images.
	batch := suite.db.NewWriteBatch()
	defer batch.Cancel()

	err := writeMappings(batch, clusterMappings)
	suite.NoError(err)
	err = writeMappings(batch, namespaceMappings)
	suite.NoError(err)
	err = writeMappings(batch, namespaceSACMappings)
	suite.NoError(err)

	err = batch.Flush()
	suite.NoError(err)

	// Run the migration.
	err = migrateSchema(suite.db)
	suite.NoError(err)

	// Clusters should point to their namespace IDs.
	postMigrationClusters, err := readMappings(suite.db, clusterBucketName)
	suite.NoError(err)
	suite.Equal(SortedKeys{
		getNamespaceKey([]byte(namespaceID1)),
		getNamespaceKey([]byte(namespaceID2)),
		getNamespaceKey([]byte(namespaceID3)),
	}, postMigrationClusters[string(getClusterKey([]byte(clusterID1)))])
	suite.Equal(SortedKeys{
		getNamespaceKey([]byte(namespaceID4)),
		getNamespaceKey([]byte(namespaceID5)),
	}, postMigrationClusters[string(getClusterKey([]byte(clusterID2)))])

	// Namespace IDs should point to their deployments and their names (SAC Bucket)
	postMigrationNamespaces, err := readMappings(suite.db, namespaceBucketName)
	suite.NoError(err)
	suite.Equal(SortedKeys{
		getDeploymentKey([]byte(deploymentID1)),
		getDeploymentKey([]byte(deploymentID2)),
		getNamespaceSACKey([]byte(namespaceName1)),
	}, postMigrationNamespaces[string(getNamespaceKey([]byte(namespaceID1)))])
	suite.Equal(SortedKeys{
		getDeploymentKey([]byte(deploymentID3)),
		getDeploymentKey([]byte(deploymentID4)),
		getNamespaceSACKey([]byte(namespaceName2)),
	}, postMigrationNamespaces[string(getNamespaceKey([]byte(namespaceID2)))])
	suite.Equal(SortedKeys{
		getDeploymentKey([]byte(deploymentID5)),
		getDeploymentKey([]byte(deploymentID6)),
		getDeploymentKey([]byte(deploymentID7)),
		getDeploymentKey([]byte(deploymentID8)),
		getNamespaceSACKey([]byte(namespaceName3)),
	}, postMigrationNamespaces[string(getNamespaceKey([]byte(namespaceID3)))])
	suite.Equal(SortedKeys{
		getDeploymentKey([]byte(deploymentID9)),
		getDeploymentKey([]byte(deploymentID10)),
		getNamespaceSACKey([]byte(namespaceName1)),
	}, postMigrationNamespaces[string(getNamespaceKey([]byte(namespaceID4)))])
	// Namespace ID 5 did not have any deployments, so it does not have deployments or a name.
	suite.Empty(postMigrationNamespaces[string(getNamespaceKey([]byte(namespaceID5)))])

	// Namespace Names (SAC Bucket) should not have any key/value pairs in the DB.
	postMigrationNamespaceSACs, err := readMappings(suite.db, namespaceSACBucketName)
	suite.NoError(err)
	suite.Empty(postMigrationNamespaceSACs)
}
