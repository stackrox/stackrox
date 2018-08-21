package store

import (
	"os"
	"sort"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stretchr/testify/suite"
)

func TestDNRIntegrationStore(t *testing.T) {
	suite.Run(t, new(DNRIntegrationStoreTestSuite))
}

type DNRIntegrationStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store Store
}

func (suite *DNRIntegrationStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store = New(db)
}

func (suite *DNRIntegrationStoreTestSuite) TeardownSuite() {
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func (suite *DNRIntegrationStoreTestSuite) TestStore() {
	integrations := []*v1.DNRIntegration{
		{
			ClusterIds: []string{"A", "B"},
		},
		{
			ClusterIds: []string{"C"},
		},
	}

	// Test Add
	for _, integration := range integrations {
		id, err := suite.store.AddDNRIntegration(integration)
		suite.Require().NoError(err)
		suite.Equal(integration.GetId(), id)
	}

	// Test Retrieval
	for _, originalIntegration := range integrations {
		retrievedIntegration, exists, err := suite.store.GetDNRIntegration(originalIntegration.GetId())
		suite.Require().NoError(err)
		suite.Require().True(exists)
		suite.Equal(originalIntegration, retrievedIntegration)
	}

	retrievedIntegrations, err := suite.store.GetDNRIntegrations(&v1.GetDNRIntegrationsRequest{ClusterId: "A"})
	suite.Require().NoError(err)
	suite.Len(retrievedIntegrations, 1)
	suite.Equal(retrievedIntegrations[0], integrations[0])

	retrievedIntegrations, err = suite.store.GetDNRIntegrations(&v1.GetDNRIntegrationsRequest{ClusterId: "C"})
	suite.Require().NoError(err)
	suite.Len(retrievedIntegrations, 1)
	suite.Equal(retrievedIntegrations[0], integrations[1])

	retrievedIntegrations, err = suite.store.GetDNRIntegrations(&v1.GetDNRIntegrationsRequest{ClusterId: "INVALID"})
	suite.Require().NoError(err)
	suite.Len(retrievedIntegrations, 0)

	sort.Slice(integrations, func(i, j int) bool {
		return integrations[i].Id < integrations[j].Id
	})

	retrievedIntegrations, err = suite.store.GetDNRIntegrations(&v1.GetDNRIntegrationsRequest{})
	suite.Require().NoError(err)
	suite.Len(retrievedIntegrations, 2)

	sort.Slice(retrievedIntegrations, func(i, j int) bool {
		return retrievedIntegrations[i].Id < retrievedIntegrations[j].Id
	})

}
