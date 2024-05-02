//go:build sql_integration

package datastore

import (
	"context"
	"math/rand"
	"testing"
	"time"

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

	resourceSelectorIncomplete = storage.ResourceSelector{
		Rules: []*storage.SelectorRule{
			{
				FieldName: "Cluster",
				Values: []*storage.RuleValue{
					{
						Value: "cluster-1",
					},
				},
			},
		},
	}

	resourceSelectorsIncomplete = []*storage.ResourceSelector{&resourceSelectorIncomplete}

	resourceCollectionIncomplete = storage.ResourceCollection{
		Id:                "b703d50e-b003-4a6a-bf1b-7ab36c9af184",
		Name:              "Incomplete",
		ResourceSelectors: resourceSelectorsIncomplete,
	}

	resourceSelectorCluster1 = storage.ResourceSelector{
		Rules: []*storage.SelectorRule{
			{
				FieldName: "Cluster",
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
				FieldName: "Cluster",
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
				FieldName: "Cluster",
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

func getRuntimeFilterMap(runtimeFilters []*storage.RuntimeFilter) map[storage.RuntimeFilterFeatures]*storage.RuntimeFilter {
	runtimeFilteringMap := make(map[storage.RuntimeFilterFeatures]*storage.RuntimeFilter)
	for _, runtimeFilter := range runtimeFilters {
		runtimeFilteringMap[runtimeFilter.Feature] = runtimeFilter
	}
	return runtimeFilteringMap
}

func (suite *RuntimeConfigurationTestSuite) compareRuntimeFilteringConfigurations(config1 *storage.RuntimeFilteringConfiguration, config2 *storage.RuntimeFilteringConfiguration) {
	runtimeFilteringMap1 := getRuntimeFilterMap(config1.RuntimeFilters)
	runtimeFilteringMap2 := getRuntimeFilterMap(config2.RuntimeFilters)

	suite.Equal(runtimeFilteringMap1, runtimeFilteringMap2)
	suite.Equal(config1.ResourceCollections, config2.ResourceCollections)
}

// TestSetRuntimeConfiguration: Writes a config to the database, reads the database
// and makes sure that the data retrieved is the same that was inserted
func (suite *RuntimeConfigurationTestSuite) TestSetRuntimeConfiguration() {
	suite.NoError(suite.datastore.SetRuntimeConfiguration(suite.hasAllCtx, runtimeFilteringConfiguration))

	fetchedRuntimeConfiguration, err := suite.datastore.GetRuntimeConfiguration(suite.hasAllCtx)
	suite.NoError(err)

	suite.compareRuntimeFilteringConfigurations(runtimeFilteringConfiguration, fetchedRuntimeConfiguration)
}

// TestSetRuntimeConfigurationNil: Attempts to write an empty config to the database.
func (suite *RuntimeConfigurationTestSuite) TestSetRuntimeConfigurationNil() {
	runtimeFilteringConfigurationNil := &storage.RuntimeFilteringConfiguration{}
	suite.NoError(suite.datastore.SetRuntimeConfiguration(suite.hasAllCtx, runtimeFilteringConfigurationNil))

	fetchedRuntimeConfiguration, err := suite.datastore.GetRuntimeConfiguration(suite.hasAllCtx)
	suite.NoError(err)

	suite.compareRuntimeFilteringConfigurations(runtimeFilteringConfigurationNil, fetchedRuntimeConfiguration)
}

// TestSetRuntimeConfigurationDefaultOnly: Writes a config to the database without any rules, reads the database
// and makes sure that the data retrieved is the same that was inserted
func (suite *RuntimeConfigurationTestSuite) TestSetRuntimeConfigurationDefaultOnly() {
	suite.NoError(suite.datastore.SetRuntimeConfiguration(suite.hasAllCtx, runtimeFilteringConfigurationDefaultOnly))

	fetchedRuntimeConfiguration, err := suite.datastore.GetRuntimeConfiguration(suite.hasAllCtx)
	suite.NoError(err)

	suite.compareRuntimeFilteringConfigurations(runtimeFilteringConfigurationDefaultOnly, fetchedRuntimeConfiguration)
}

// TestSetRuntimeConfigurationIncomplete: Some fields are nil
func (suite *RuntimeConfigurationTestSuite) TestSetRuntimeConfigurationIncomplete() {
	runtimeFilteringConfigurationIncomplete := &storage.RuntimeFilteringConfiguration{
		ResourceCollections: []*storage.ResourceCollection{&resourceCollectionIncomplete},
	}

	suite.NoError(suite.datastore.SetRuntimeConfiguration(suite.hasAllCtx, runtimeFilteringConfigurationIncomplete))

	fetchedRuntimeConfiguration, err := suite.datastore.GetRuntimeConfiguration(suite.hasAllCtx)
	suite.NoError(err)

	suite.compareRuntimeFilteringConfigurations(runtimeFilteringConfigurationIncomplete, fetchedRuntimeConfiguration)
}

func randomString(length int) string {
    //rand.Seed(time.Now().UnixNano())

    const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

    result := make([]byte, length)

    for i := range result {
        result[i] = charset[rand.Intn(len(charset))]
    }

    return string(result)
}


func makeRandomCollection() storage.ResourceCollection {
	resourceSelector := storage.ResourceSelector{
		Rules: []*storage.SelectorRule{
			{
				FieldName: "Cluster",
				Operator:  storage.BooleanOperator_OR,
				Values: []*storage.RuleValue{
					{
						Value:     randomString(10),
						MatchType: storage.MatchType_EXACT,
					},
				},
			},
			{
				FieldName: "Namespace",
				Operator:  storage.BooleanOperator_OR,
				Values: []*storage.RuleValue{
					{
						Value:     randomString(10),
						MatchType: storage.MatchType_EXACT,
					},
				},
			},
		},
	}
	resourceSelectors := []*storage.ResourceSelector{&resourceSelector}
	resourceCollection := storage.ResourceCollection{
		Id:             randomString(16),
		Name:		randomString(16),
		ResourceSelectors: resourceSelectors,
	}

	return resourceCollection
}

func makeRandomCollections(ncollections int) []*storage.ResourceCollection {
	resourceCollections := make([]*storage.ResourceCollection, ncollections)

	for i := 0; i < ncollections; i++ {
		resourceCollection := makeRandomCollection()
		resourceCollections[i] = &resourceCollection;
	}

	return resourceCollections
}

func makeRandomRules(collections []*storage.ResourceCollection) []*storage.RuntimeFilter_RuntimeFilterRule {
	ncollections := len(collections)
	rules := make([]*storage.RuntimeFilter_RuntimeFilterRule, ncollections)
	for i := 0; i < ncollections; i++ {
		runtimeFilterRule := storage.RuntimeFilter_RuntimeFilterRule{
			ResourceCollectionId: collections[i].Id,
			Status:			randomString(10),
		}
		rules[i] = &runtimeFilterRule;
	}

	return rules
}

func makeRandomRuntimeFilters(collections []*storage.ResourceCollection) []*storage.RuntimeFilter {
	//runtimeFilters := make([]*storage.RuntimeFilter, len(storage.RuntimeFilterFeatures_name))
	runtimeFilters := make([]*storage.RuntimeFilter, 0)
	//for _, feature := range storage.RuntimeFilterFeatures_value {
	features := []storage.RuntimeFilterFeatures{
		storage.RuntimeFilterFeatures_EXTERNAL_IPS,
		storage.RuntimeFilterFeatures_PROCESSES,
		storage.RuntimeFilterFeatures_NETWORK_CONNECTIONS,
		storage.RuntimeFilterFeatures_LISTENING_ENDPOINTS,
	}

	for i := 0; i < 4; i++ {
		rules := makeRandomRules(collections)
		runtimeFilter := storage.RuntimeFilter{
			Feature:       features[i],
			DefaultStatus: randomString(10),
			Rules:         rules,
		}
		//runtimeFilters[feature] = &runtimeFilter
		runtimeFilters = append(runtimeFilters, &runtimeFilter)
	}

	return runtimeFilters
}

func makeRandomConfiguration(ncollections int) *storage.RuntimeFilteringConfiguration {
	resourceCollections := makeRandomCollections(ncollections)
	runtimeFilters := makeRandomRuntimeFilters(resourceCollections)

	runtimeFilteringConfiguration = &storage.RuntimeFilteringConfiguration{
		RuntimeFilters:      runtimeFilters,
		ResourceCollections: resourceCollections,
	}

	return runtimeFilteringConfiguration
}

func (suite *RuntimeConfigurationTestSuite) TestSetRuntimeConfigurationSpeed() {
	runtimeFilteringConfigurationLarge := makeRandomConfiguration(100000)

	start := time.Now()
	suite.NoError(suite.datastore.SetRuntimeConfiguration(suite.hasAllCtx, runtimeFilteringConfigurationLarge))
	setTime := time.Since(start)
	log.Infof("setTime= %+v", setTime)

	getStart := time.Now()
	fetchedRuntimeConfiguration, err := suite.datastore.GetRuntimeConfiguration(suite.hasAllCtx)
	suite.NoError(err)
	getTime := time.Since(getStart)
	log.Infof("getTime= %+v", getTime)

	elapsedTime := time.Since(start)
	log.Infof("elapsedTime= %+v", elapsedTime)

	suite.compareRuntimeFilteringConfigurations(runtimeFilteringConfigurationLarge, fetchedRuntimeConfiguration)
}
