package blevesearch

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/fixtures"
	"github.com/stretchr/testify/suite"
)

func TestSearch(t *testing.T) {
	suite.Run(t, new(SearchTestSuite))
}

type SearchTestSuite struct {
	suite.Suite
	*Indexer
}

func (suite *SearchTestSuite) SetupSuite() {
	indexer, err := NewIndexer()
	suite.Require().NoError(err)

	suite.Indexer = indexer
	suite.NoError(suite.Indexer.AddAlert(fixtures.GetAlert()))
}

func (suite *SearchTestSuite) TeardownSuite() {
	suite.Indexer.Close()
}

func (suite *SearchTestSuite) TestValuesToDisjunctionQuery() {
	values := &v1.ParsedSearchRequest_Values{
		Values: []string{
			"blah",
			"docker.io",
		},
	}
	query := valuesToDisjunctionQuery("policy.image_policy.image_name.registry", values)
	results, err := runQuery(query, suite.alertIndex)
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal("Alert1", results[0].ID)

	values = &v1.ParsedSearchRequest_Values{
		Values: []string{
			"blah",
		},
	}
	query = valuesToDisjunctionQuery("policy.image_policy.image_name.registry", values)
	results, err = runQuery(query, suite.alertIndex)
	suite.NoError(err)
	suite.Empty(results)
}

func (suite *SearchTestSuite) TestFieldsToQuery() {
	fieldMap := map[string]*v1.ParsedSearchRequest_Values{
		"policy.image_policy.image_name.registry": {
			Values: []string{
				"blah",
				"docker.io",
			},
		},
		"policy.image_policy.image_name.namespace": {
			Values: []string{
				"stackrox",
			},
		},
	}
	query := fieldsToQuery(fieldMap, alertObjectMap)
	results, err := runQuery(query, suite.alertIndex)
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal("Alert1", results[0].ID)

	// Add a field that does not exist and this should fail
	fieldMap["blah"] = &v1.ParsedSearchRequest_Values{
		Values: []string{"blah"},
	}
	query = fieldsToQuery(fieldMap, alertObjectMap)
	results, err = runQuery(query, suite.alertIndex)
	suite.NoError(err)
	suite.Empty(results)
}

func (suite *SearchTestSuite) TestRunRequest() {
	// No scopes search
	request := &v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			"id": {
				Values: []string{"Alert1"},
			},
		},
	}
	results, err := runSearchRequest(request, suite.alertIndex, scopeToAlertQuery, alertObjectMap)
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal("Alert1", results[0].ID)

	// No fields search
	request = &v1.ParsedSearchRequest{
		Scopes: []*v1.Scope{
			{
				Cluster:   "prod cluster",
				Namespace: "stackrox",
			},
		},
	}
	results, err = runSearchRequest(request, suite.alertIndex, scopeToAlertQuery, alertObjectMap)
	suite.NoError(err)
	suite.Len(results, 1)

	// Combined search with success
	request = &v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			"id": {
				Values: []string{"Alert1"},
			},
			"policy.name": {
				Values: []string{"vulnerable"},
			},
		},
		Scopes: []*v1.Scope{
			{
				Cluster:   "prod cluster",
				Namespace: "stackrox",
			},
		},
	}
	results, err = runSearchRequest(request, suite.alertIndex, scopeToAlertQuery, alertObjectMap)
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal("Alert1", results[0].ID)

	// Combined search with failure
	request = &v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			"id": {
				Values: []string{"NoID"},
			},
			"policy.name": {
				Values: []string{"vulnerable"},
			},
		},
		Scopes: []*v1.Scope{
			{
				Cluster:   "prod cluster",
				Namespace: "stackrox",
			},
		},
	}
	results, err = runSearchRequest(request, suite.alertIndex, scopeToAlertQuery, alertObjectMap)
	suite.NoError(err)
	suite.Len(results, 0)
}

func (suite *SearchTestSuite) TestRunQuery() {
	query := newFuzzyQuery("id", "Alert1", fuzzyPrefix)
	results, err := runQuery(query, suite.alertIndex)
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal("Alert1", results[0].ID)

	query = newFuzzyQuery("id", "blahblah", fuzzyPrefix)
	results, err = runQuery(query, suite.alertIndex)
	suite.NoError(err)
	suite.Len(results, 0)
}

func (suite *SearchTestSuite) TestTransformFields() {
	fields := map[string]*v1.ParsedSearchRequest_Values{
		"image.name.sha": {
			Values: []string{"sha"},
		},
		"blah": {
			Values: []string{"blah"},
		},
		"alert.deployment.name": {
			Values: []string{"name"},
		},
	}

	expectedFields := map[string]*v1.ParsedSearchRequest_Values{
		"deployment.containers.image.name.sha": {
			Values: []string{"sha"},
		},
		"blah": {
			Values: []string{"blah"},
		},
		"deployment.name": {
			Values: []string{"name"},
		},
	}
	suite.Equal(expectedFields, transformFields(fields, alertObjectMap))
}
