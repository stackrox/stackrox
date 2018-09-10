package search

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestParseRawQuery(t *testing.T) {
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
							MatchFieldQuery: &v1.MatchFieldQuery{Field: Category.String(), Value: "field2"},
						},
					},
				}},
			},
		}},
	}
	actualQuery, err := ParseRawQuery(query)
	assert.NoError(t, err)
	assert.Equal(t, expectedQuery, actualQuery)

	query = fmt.Sprintf("%s:field1,field12 + Has:rawstuff+ %s:field2", DeploymentName, Category)

	expectedQuery = &v1.Query{
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
						Query: &v1.BaseQuery_StringQuery{
							StringQuery: &v1.StringQuery{Query: "rawstuff"},
						},
					},
				}},
				{Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchFieldQuery{
							MatchFieldQuery: &v1.MatchFieldQuery{Field: Category.String(), Value: "field2"},
						},
					},
				}},
			},
		}},
	}
	actualQuery, err = ParseRawQuery(query)
	assert.NoError(t, err)
	assert.Equal(t, expectedQuery, actualQuery)

	_, err = ParseRawQuery("")
	assert.Error(t, err)
	actualQuery, err = ParseRawQueryOrEmpty("")
	assert.NoError(t, err)
	assert.Equal(t, EmptyQuery(), actualQuery)

	// An invalid query should always return an error.
	query = "INVALIDQUERY"
	_, err = ParseRawQuery(query)
	assert.Error(t, err)
	_, err = ParseRawQueryOrEmpty(query)
	assert.Error(t, err)
}
