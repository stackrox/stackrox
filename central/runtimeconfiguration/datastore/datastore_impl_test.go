//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	runtimeStore "github.com/stackrox/rox/central/runtimeconfiguration/store"
	postgresStore "github.com/stackrox/rox/central/runtimeconfiguration/store/postgres"
	runtimeCollectionsStore "github.com/stackrox/rox/central/runtimeconfigurationcollection/store/postgres"
	"github.com/stackrox/rox/generated/storage"

	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

func TestRuntimeContigurationDataStore(t *testing.T) {
	suite.Run(t, new(RuntimeConfigurationTestSuite))
}

type RuntimeConfigurationTestSuite struct {
	suite.Suite
	datastore DataStore
	store     runtimeStore.Store

	postgres *pgtest.TestPostgres

	hasAllCtx context.Context
}

func (suite *RuntimeConfigurationTestSuite) SetupSuite() {
	suite.hasAllCtx = sac.WithAllAccess(context.Background())
}

func (suite *RuntimeConfigurationTestSuite) SetupTest() {
	suite.postgres = pgtest.ForT(suite.T())
	suite.store = postgresStore.New(suite.postgres.DB)
	rcStore := runtimeCollectionsStore.New(suite.postgres.DB)

	suite.datastore = New(suite.store, rcStore, suite.postgres)
}

func (suite *RuntimeConfigurationTestSuite) TearDownTest() {
	suite.postgres.Teardown(suite.T())
}

// TestSetRuntimeConfiguration: Writes a config to the database, reads the database
// and makes sure that the data retrieved is the same that was inserted
func (suite *RuntimeConfigurationTestSuite) TestSetRuntimeConfiguration() {

	runtimeFilterRule := storage.RuntimeFilter_RuntimeFilterRule{
		ResourceCollectionId: "abcd",
		Status:               "off",
	}

	rules := []*storage.RuntimeFilter_RuntimeFilterRule{&runtimeFilterRule}

	runtimeFilter := storage.RuntimeFilter{
		Feature:       storage.RuntimeFilterFeatures_PROCESSES,
		DefaultStatus: "on",
		Rules:         rules,
	}

	resourceSelector := storage.ResourceSelector{
		Rules: []*storage.SelectorRule{
			{
				FieldName: "Namespace",
				Operator:  storage.BooleanOperator_OR,
				Values: []*storage.RuleValue{
					{
						Value:     "webapp",
						MatchType: storage.MatchType_EXACT,
					},
				},
			},
		},
	}

	resourceSelectors := []*storage.ResourceSelector{&resourceSelector}

	resourceCollection := storage.ResourceCollection{
		Id:                "abcd",
		Name:              "Fake collection",
		ResourceSelectors: resourceSelectors,
	}

	runtimeFilters := []*storage.RuntimeFilter{&runtimeFilter}
	resourceCollections := []*storage.ResourceCollection{&resourceCollection}

	runtimeFilteringConfiguration := &storage.RuntimeFilteringConfiguration{
		RuntimeFilters:      runtimeFilters,
		ResourceCollections: resourceCollections,
	}

	suite.NoError(suite.datastore.SetRuntimeConfiguration(suite.hasAllCtx, runtimeFilteringConfiguration))

	fetchedRuntimeConfiguration, err := suite.datastore.GetRuntimeConfiguration(suite.hasAllCtx)
	suite.NoError(err)

	suite.Equal(runtimeFilteringConfiguration, fetchedRuntimeConfiguration)
}
