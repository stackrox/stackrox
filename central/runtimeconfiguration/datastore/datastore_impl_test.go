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

var (
	runtimeFilterRuleCluster1 = storage.RuntimeFilter_RuntimeFilterRule{
		ResourceCollectionId: "abcd",
		Status:               "off",
	}

	runtimeFilterRuleWebappAndMarketing = storage.RuntimeFilter_RuntimeFilterRule{
		ResourceCollectionId: "qwerty",
		Status:               "on",
	}

	runtimeFilterRuleMarketingDepartment = storage.RuntimeFilter_RuntimeFilterRule{
		ResourceCollectionId: "asdf",
		Status:               "off",
	}

	rules = []*storage.RuntimeFilter_RuntimeFilterRule{
		&runtimeFilterRuleCluster1,
		&runtimeFilterRuleWebappAndMarketing,
		&runtimeFilterRuleMarketingDepartment,
	}

	runtimeFilterExternalIPs = storage.RuntimeFilter{
		Feature:       storage.RuntimeFilterFeatures_EXTERNAL_IPS,
		DefaultStatus: "off",
		Rules:         rules,
	}

	runtimeFilterProcess = storage.RuntimeFilter{
		Feature:       storage.RuntimeFilterFeatures_PROCESSES,
		DefaultStatus: "on",
	}

	runtimeFilterNetworkConnections = storage.RuntimeFilter{
		Feature:       storage.RuntimeFilterFeatures_NETWORK_CONNECTIONS,
		DefaultStatus: "on",
	}

	runtimeFilterListeningEndpoints = storage.RuntimeFilter{
		Feature:       storage.RuntimeFilterFeatures_LISTENING_ENDPOINTS,
		DefaultStatus: "on",
	}

	resourceSelectorCluster1 = storage.ResourceSelector{
		Rules: []*storage.SelectorRule{
			{
				FieldName: "Clster",
				Operator:  storage.BooleanOperator_OR,
				Values: []*storage.RuleValue{
					{
						Value:     "cluster-1",
						MatchType: storage.MatchType_EXACT,
					},
				},
			},
		},
	}

	resourceSelectorWebappAndMarketing = storage.ResourceSelector{
		Rules: []*storage.SelectorRule{
			{
				FieldName: "Clster",
				Operator:  storage.BooleanOperator_OR,
				Values: []*storage.RuleValue{
					{
						Value:     "cluster-1",
						MatchType: storage.MatchType_EXACT,
					},
				},
			},
			{
				FieldName: "Namespace",
				Operator:  storage.BooleanOperator_OR,
				Values: []*storage.RuleValue{
					{
						Value:     "webapp",
						MatchType: storage.MatchType_EXACT,
					},
					{
						Value:     "marketing.*",
						MatchType: storage.MatchType_REGEX,
					},
				},
			},
		},
	}

	resourceSelectorMarketingDepartment = storage.ResourceSelector{
		Rules: []*storage.SelectorRule{
			{
				FieldName: "Clster",
				Operator:  storage.BooleanOperator_OR,
				Values: []*storage.RuleValue{
					{
						Value:     "cluster-1",
						MatchType: storage.MatchType_EXACT,
					},
				},
			},
			{
				FieldName: "Namespace",
				Operator:  storage.BooleanOperator_OR,
				Values: []*storage.RuleValue{
					{
						Value:     "marketing-department",
						MatchType: storage.MatchType_EXACT,
					},
				},
			},
		},
	}

	resourceSelectorsCluster1            = []*storage.ResourceSelector{&resourceSelectorCluster1}
	resourceSelectorsWebappAndMarketing  = []*storage.ResourceSelector{&resourceSelectorWebappAndMarketing}
	resourceSelectorsMarketingDepartment = []*storage.ResourceSelector{&resourceSelectorMarketingDepartment}

	resourceCollectionCluster1 = storage.ResourceCollection{
		Id:                "abcd",
		Name:              "Cluster 1",
		ResourceSelectors: resourceSelectorsCluster1,
	}

	resourceCollectionWebappAndMarketing = storage.ResourceCollection{
		Id:                "qwerty",
		Name:              "Webapp and marketing",
		ResourceSelectors: resourceSelectorsWebappAndMarketing,
	}

	resourceCollectionMarketingDepartment = storage.ResourceCollection{
		Id:                "asdf",
		Name:              "Marketing Department",
		ResourceSelectors: resourceSelectorsMarketingDepartment,
	}

	runtimeFilters = []*storage.RuntimeFilter{
		&runtimeFilterExternalIPs,
		&runtimeFilterProcess,
		&runtimeFilterNetworkConnections,
		&runtimeFilterListeningEndpoints,
	}

	runtimeFiltersDefaultOnly = []*storage.RuntimeFilter{&runtimeFilterProcess}

	resourceCollections = []*storage.ResourceCollection{
		&resourceCollectionCluster1,
		&resourceCollectionWebappAndMarketing,
		&resourceCollectionMarketingDepartment,
	}

	runtimeFilteringConfiguration = &storage.RuntimeFilteringConfiguration{
		RuntimeFilters:      runtimeFilters,
		ResourceCollections: resourceCollections,
	}

	runtimeFilteringConfigurationDefaultOnly = &storage.RuntimeFilteringConfiguration{
		RuntimeFilters: runtimeFiltersDefaultOnly,
	}
)

// TestSetRuntimeConfiguration: Writes a config to the database, reads the database
// and makes sure that the data retrieved is the same that was inserted
func (suite *RuntimeConfigurationTestSuite) TestSetRuntimeConfiguration() {
	suite.NoError(suite.datastore.SetRuntimeConfiguration(suite.hasAllCtx, runtimeFilteringConfiguration))

	fetchedRuntimeConfiguration, err := suite.datastore.GetRuntimeConfiguration(suite.hasAllCtx)
	suite.NoError(err)

	suite.Equal(runtimeFilteringConfiguration, fetchedRuntimeConfiguration)
}

// TestSetRuntimeConfigurationNil: Attempts to write an empty config to the database.
func (suite *RuntimeConfigurationTestSuite) TestSetRuntimeConfigurationNil() {
	runtimeFilteringConfigurationNil := &storage.RuntimeFilteringConfiguration{}
	suite.NoError(suite.datastore.SetRuntimeConfiguration(suite.hasAllCtx, runtimeFilteringConfigurationNil))

	fetchedRuntimeConfiguration, err := suite.datastore.GetRuntimeConfiguration(suite.hasAllCtx)
	suite.NoError(err)

	suite.Equal(runtimeFilteringConfigurationNil, fetchedRuntimeConfiguration)
}

// TestSetRuntimeConfigurationDefaultOnly: Writes a config to the database without any rules, reads the database
// and makes sure that the data retrieved is the same that was inserted
func (suite *RuntimeConfigurationTestSuite) TestSetRuntimeConfigurationDefaultOnly() {
	suite.NoError(suite.datastore.SetRuntimeConfiguration(suite.hasAllCtx, runtimeFilteringConfigurationDefaultOnly))

	fetchedRuntimeConfiguration, err := suite.datastore.GetRuntimeConfiguration(suite.hasAllCtx)
	suite.NoError(err)

	suite.Equal(runtimeFilteringConfigurationDefaultOnly, fetchedRuntimeConfiguration)
}
