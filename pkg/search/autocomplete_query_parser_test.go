package search

import (
	"fmt"
	"testing"

	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestParseAutocompleteQuery(t *testing.T) {
	query := fmt.Sprintf("%s:field1,field12+%s:field2", DeploymentName, Category)
	expectedQuery := &v1.Query{
		Query: &v1.Query_Conjunction{Conjunction: &v1.ConjunctionQuery{
			Queries: []*v1.Query{
				{Query: &v1.Query_Disjunction{Disjunction: &v1.DisjunctionQuery{
					Queries: []*v1.Query{
						{Query: &v1.Query_BaseQuery{
							BaseQuery: &v1.BaseQuery{
								Query: &v1.BaseQuery_MatchFieldQuery{
									MatchFieldQuery: &v1.MatchFieldQuery{Field: DeploymentName.String(), Value: "field1"},
								},
							},
						}},
						{Query: &v1.Query_BaseQuery{
							BaseQuery: &v1.BaseQuery{
								Query: &v1.BaseQuery_MatchFieldQuery{
									MatchFieldQuery: &v1.MatchFieldQuery{Field: DeploymentName.String(), Value: "field12"},
								},
							},
						}},
					},
				}}},
				{Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchFieldQuery{
							MatchFieldQuery: &v1.MatchFieldQuery{Field: Category.String(), Value: "field2", Highlight: true},
						},
					},
				}},
			},
		}},
	}
	expectedKey := Category.String()

	var actualKey string
	actualQuery, actualKey, err := autocompleteQueryParser{}.parse(query)
	assert.NoError(t, err)
	assert.Equal(t, expectedKey, actualKey)
	assert.Equal(t, expectedQuery, actualQuery)

	_, _, err = autocompleteQueryParser{}.parse("")
	assert.Error(t, err)

	// An invalid query should always return an error.
	query = "INVALIDQUERY"
	_, _, err = autocompleteQueryParser{}.parse(query)
	assert.Error(t, err)
}
