package search

import (
	"fmt"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestParseRawQuery(t *testing.T) {
	query := fmt.Sprintf("%s:field1,field12+%s:field2", DeploymentName, Category)
	expectedQuery := &v1.Query{
		Query: &v1.Query_Conjunction{Conjunction: &v1.ConjunctionQuery{
			Queries: []*v1.Query{
				{Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchFieldQuery{
							MatchFieldQuery: &v1.MatchFieldQuery{Field: Category.String(), Value: "field2"},
						},
					},
				}},
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
			},
		}},
	}
	actualQuery, err := generalQueryParser{}.parse(query)
	assert.NoError(t, err)
	assert.Equal(t, expectedQuery, actualQuery)

	query = fmt.Sprintf("%s:field1,field12 + %s:field2", DeploymentName, Category)

	expectedQuery = &v1.Query{
		Query: &v1.Query_Conjunction{Conjunction: &v1.ConjunctionQuery{
			Queries: []*v1.Query{
				{Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchFieldQuery{
							MatchFieldQuery: &v1.MatchFieldQuery{Field: Category.String(), Value: "field2"},
						},
					},
				}},
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
			},
		}},
	}
	actualQuery, err = generalQueryParser{}.parse(query)
	assert.NoError(t, err)
	assert.Equal(t, expectedQuery, actualQuery)

	_, err = generalQueryParser{}.parse("")
	assert.Error(t, err)
	actualQuery, err = generalQueryParser{MatchAllIfEmpty: true}.parse("")
	assert.NoError(t, err)
	assert.Equal(t, EmptyQuery(), actualQuery)

	// An invalid query should return an error.
	query = "INVALIDQUERY"
	_, err = generalQueryParser{}.parse(query)
	assert.Error(t, err)
}
