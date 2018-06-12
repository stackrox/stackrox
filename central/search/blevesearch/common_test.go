package blevesearch

import (
	"testing"

	"io/ioutil"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func NewTmpIndexer() (*Indexer, error) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}
	return NewIndexer(dir)
}

func TestSearch(t *testing.T) {
	suite.Run(t, new(SearchTestSuite))
}

type SearchTestSuite struct {
	suite.Suite
	*Indexer
}

func (suite *SearchTestSuite) SetupSuite() {
	indexer, err := NewTmpIndexer()
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
			"vuln",
		},
	}
	query := valuesToDisjunctionQuery("alert.policy.name", values)
	results, err := runQuery(query, suite.globalIndex)
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal("Alert1", results[0].ID)

	values = &v1.ParsedSearchRequest_Values{
		Values: []string{
			"blah",
		},
	}
	query = valuesToDisjunctionQuery("alert.policy.name", values)
	results, err = runQuery(query, suite.globalIndex)
	suite.NoError(err)
	suite.Empty(results)
}

func (suite *SearchTestSuite) TestFieldsToQuery() {
	fieldMap := map[string]*v1.ParsedSearchRequest_Values{
		"policy.name": {
			Field: &v1.SearchField{
				FieldPath: "policy.name",
				Type:      v1.SearchDataType_SEARCH_STRING,
			},
			Values: []string{
				"blah",
				"vuln",
			},
		},
		"policy.description": {
			Field: &v1.SearchField{
				FieldPath: "alert.policy.description",
				Type:      v1.SearchDataType_SEARCH_STRING,
			},
			Values: []string{
				"alert",
			},
		},
	}
	query, err := fieldsToQuery(fieldMap, alertObjectMap)
	suite.NoError(err)

	results, err := runQuery(query, suite.globalIndex)
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
	results, err = runQuery(query, suite.globalIndex)
	suite.NoError(err)
	suite.Empty(results)
}

func (suite *SearchTestSuite) TestRunRequest() {
	// No scopes search
	request := &v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			"policy.name": {
				Field: &v1.SearchField{
					FieldPath: "alert.policy.name",
					Type:      v1.SearchDataType_SEARCH_STRING,
				},
				Values: []string{"vuln"},
			},
		},
	}
	results, err := runSearchRequest(v1.SearchCategory_ALERTS.String(), request, suite.globalIndex, scopeToAlertQuery, alertObjectMap)
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
	results, err = runSearchRequest(v1.SearchCategory_ALERTS.String(), request, suite.globalIndex, scopeToAlertQuery, alertObjectMap)
	suite.NoError(err)
	suite.Len(results, 1)

	// Combined search with success
	request = &v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			"policy.description": {
				Field: &v1.SearchField{
					FieldPath: "policy.description",
					Type:      v1.SearchDataType_SEARCH_STRING,
				},
				Values: []string{"alert"},
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
	results, err = runSearchRequest(v1.SearchCategory_ALERTS.String(), request, suite.globalIndex, scopeToAlertQuery, alertObjectMap)
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal("Alert1", results[0].ID)

	// Combined search with failure
	request = &v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			"alert.id": {
				Field: &v1.SearchField{
					FieldPath: "id",
					Type:      v1.SearchDataType_SEARCH_STRING,
				},
				Values: []string{"NoID"},
			},
			"alert.policy.name": {
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
	results, err = runSearchRequest(v1.SearchCategory_ALERTS.String(), request, suite.globalIndex, scopeToAlertQuery, alertObjectMap)
	suite.NoError(err)
	suite.Len(results, 0)
}

func (suite *SearchTestSuite) TestRunQuery() {
	query := newPrefixQuery("alert.policy.name", "vuln")
	results, err := runQuery(query, suite.globalIndex)
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal("Alert1", results[0].ID)

	query = newPrefixQuery("id", "blahblah")
	results, err = runQuery(query, suite.globalIndex)
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
		"deployment.name": {
			Field: &v1.SearchField{
				FieldPath: "deployment.name",
				Type:      v1.SearchDataType_SEARCH_STRING,
			},
			Values: []string{"name"},
		},
	}

	expectedFields := map[string]*v1.ParsedSearchRequest_Values{
		"alert.deployment.containers.image.name.sha": {
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
				FieldPath: "deployment.name",
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
