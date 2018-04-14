package blevesearch

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/fixtures"
	"github.com/stretchr/testify/assert"
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
			Field: &v1.SearchField{
				FieldPath: "policy.image_policy.image_name.registry",
				Type:      v1.SearchDataType_SEARCH_STRING,
			},
			Values: []string{
				"blah",
				"docker.io",
			},
		},
		"policy.image_policy.image_name.namespace": {
			Field: &v1.SearchField{
				FieldPath: "policy.image_policy.image_name.namespace",
				Type:      v1.SearchDataType_SEARCH_STRING,
			},
			Values: []string{
				"stackrox",
			},
		},
	}
	query, err := fieldsToQuery(fieldMap, alertObjectMap)
	suite.NoError(err)

	results, err := runQuery(query, suite.alertIndex)
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal("Alert1", results[0].ID)

	// Add a field that does not exist and this should fail
	fieldMap["blah"] = &v1.ParsedSearchRequest_Values{
		Field: &v1.SearchField{
			FieldPath: "blah",
			Type:      v1.SearchDataType_SEARCH_STRING,
		},
		Values: []string{"blah"},
	}
	query, err = fieldsToQuery(fieldMap, alertObjectMap)
	suite.NoError(err)
	results, err = runQuery(query, suite.alertIndex)
	suite.NoError(err)
	suite.Empty(results)
}

func (suite *SearchTestSuite) TestRunRequest() {
	// No scopes search
	request := &v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			"id": {
				Field: &v1.SearchField{
					FieldPath: "id",
					Type:      v1.SearchDataType_SEARCH_STRING,
				},
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
				Field: &v1.SearchField{
					FieldPath: "id",
					Type:      v1.SearchDataType_SEARCH_STRING,
				},
				Values: []string{"Alert1"},
			},
			"policy.name": {
				Field: &v1.SearchField{
					FieldPath: "policy.name",
					Type:      v1.SearchDataType_SEARCH_STRING,
				},
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
				Field: &v1.SearchField{
					FieldPath: "id",
					Type:      v1.SearchDataType_SEARCH_STRING,
				},
				Values: []string{"NoID"},
			},
			"policy.name": {
				Field: &v1.SearchField{
					FieldPath: "policy.name",
					Type:      v1.SearchDataType_SEARCH_STRING,
				},
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
	query := newPrefixQuery("id", "Alert1")
	results, err := runQuery(query, suite.alertIndex)
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal("Alert1", results[0].ID)

	query = newPrefixQuery("id", "blahblah")
	results, err = runQuery(query, suite.alertIndex)
	suite.NoError(err)
	suite.Len(results, 0)
}

func (suite *SearchTestSuite) TestTransformFields() {
	fields := map[string]*v1.ParsedSearchRequest_Values{
		"image.name.sha": {
			Field: &v1.SearchField{
				FieldPath: "image.name.sha",
				Type:      v1.SearchDataType_SEARCH_STRING,
			},
			Values: []string{"sha"},
		},
		"blah": {
			Field: &v1.SearchField{
				FieldPath: "blah",
				Type:      v1.SearchDataType_SEARCH_STRING,
			},
			Values: []string{"blah"},
		},
		"alert.deployment.name": {
			Field: &v1.SearchField{
				FieldPath: "alert.deployment.name",
				Type:      v1.SearchDataType_SEARCH_STRING,
			},
			Values: []string{"name"},
		},
	}

	expectedFields := map[string]*v1.ParsedSearchRequest_Values{
		"deployment.containers.image.name.sha": {
			Field: &v1.SearchField{
				FieldPath: "image.name.sha",
				Type:      v1.SearchDataType_SEARCH_STRING,
			},
			Values: []string{"sha"},
		},
		"blah": {
			Field: &v1.SearchField{
				FieldPath: "blah",
				Type:      v1.SearchDataType_SEARCH_STRING,
			},
			Values: []string{"blah"},
		},
		"deployment.name": {
			Field: &v1.SearchField{
				FieldPath: "alert.deployment.name",
				Type:      v1.SearchDataType_SEARCH_STRING,
			},
			Values: []string{"name"},
		},
	}
	suite.Equal(expectedFields, transformFields(fields, alertObjectMap))
}

func TestParseNumericValue(t *testing.T) {
	t.Parallel()
	type expected struct {
		min       *float64
		max       *float64
		inclusive *bool
		hasError  bool
	}
	tests := []struct {
		input        string
		expectedVals expected
	}{
		{
			input: "<=3",
			expectedVals: expected{
				min:       nil,
				max:       floatPtr(float64(3)),
				inclusive: boolPtr(true),
				hasError:  false,
			},
		},
		{
			input: "<3",
			expectedVals: expected{
				min:       nil,
				max:       floatPtr(float64(3)),
				inclusive: boolPtr(false),
				hasError:  false,
			},
		},
		{
			input: ">=3",
			expectedVals: expected{
				min:       floatPtr(float64(3)),
				max:       nil,
				inclusive: boolPtr(true),
				hasError:  false,
			},
		},
		{
			input: ">3",
			expectedVals: expected{
				min:       floatPtr(float64(3)),
				max:       nil,
				inclusive: boolPtr(false),
				hasError:  false,
			},
		},
		{
			input: "3",
			expectedVals: expected{
				min:       floatPtr(float64(3)),
				max:       floatPtr(float64(3)),
				inclusive: boolPtr(true),
				hasError:  false,
			},
		},
		{
			input: ">=h",
			expectedVals: expected{
				hasError: true,
			},
		},
		{
			input: "=>3",
			expectedVals: expected{
				hasError: true,
			},
		},
	}

	for _, test := range tests {
		min, max, inclusive, err := parseNumericValue(test.input)
		if test.expectedVals.hasError {
			assert.Error(t, err)
			continue
		}
		assert.Equal(t, test.expectedVals.min, min)
		assert.Equal(t, test.expectedVals.max, max)
		assert.Equal(t, test.expectedVals.inclusive, inclusive)
	}

}
