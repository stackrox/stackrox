package cveedge

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
)

func TestGetCVEEdgeQuery(t *testing.T) {
	query := &v1.Query{
		Query: &v1.Query_Conjunction{Conjunction: &v1.ConjunctionQuery{
			Queries: []*v1.Query{
				{Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchFieldQuery{
							MatchFieldQuery: &v1.MatchFieldQuery{Field: search.Fixable.String(), Value: "true"},
						},
					},
				}},
				{Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchFieldQuery{
							MatchFieldQuery: &v1.MatchFieldQuery{Field: search.ClusterID.String(), Value: "cluster1"},
						},
					},
				}},
			},
		}},
	}

	expectedQuery := &v1.Query{
		Query: &v1.Query_Conjunction{Conjunction: &v1.ConjunctionQuery{
			Queries: []*v1.Query{
				{Query: &v1.Query_Disjunction{
					Disjunction: &v1.DisjunctionQuery{
						Queries: []*v1.Query{
							{Query: &v1.Query_BaseQuery{
								BaseQuery: &v1.BaseQuery{
									Query: &v1.BaseQuery_MatchFieldQuery{
										MatchFieldQuery: &v1.MatchFieldQuery{Field: search.Fixable.String(), Value: "true"},
									},
								},
							}},
							{Query: &v1.Query_BaseQuery{
								BaseQuery: &v1.BaseQuery{
									Query: &v1.BaseQuery_MatchFieldQuery{
										MatchFieldQuery: &v1.MatchFieldQuery{Field: search.ClusterCVEFixable.String(), Value: "true"},
									},
								},
							}},
						},
					},
				}},
				{Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchFieldQuery{
							MatchFieldQuery: &v1.MatchFieldQuery{Field: search.ClusterID.String(), Value: "cluster1"},
						},
					},
				}},
			},
		}},
	}

	getCVEEdgeQuery(query)
	assert.Equal(t, expectedQuery, query)

}
