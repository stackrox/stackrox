package search

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestValidateQuery(t *testing.T) {
	optionsMap := Walk(v1.SearchCategory_IMAGES, "derp", &storage.Image{})

	query := &v1.Query{
		Query: &v1.Query_Conjunction{Conjunction: &v1.ConjunctionQuery{
			Queries: []*v1.Query{
				{Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchFieldQuery{
							MatchFieldQuery: &v1.MatchFieldQuery{Field: CVE.String(), Value: "cveId"},
						},
					},
				}},
				{Query: &v1.Query_Disjunction{Disjunction: &v1.DisjunctionQuery{
					Queries: []*v1.Query{
						{Query: &v1.Query_BaseQuery{
							BaseQuery: &v1.BaseQuery{
								Query: &v1.BaseQuery_MatchFieldQuery{
									MatchFieldQuery: &v1.MatchFieldQuery{Field: DeploymentName.String(), Value: "depname"},
								},
							},
						}},
						{Query: &v1.Query_BaseQuery{
							BaseQuery: &v1.BaseQuery{
								Query: &v1.BaseQuery_MatchFieldQuery{
									MatchFieldQuery: &v1.MatchFieldQuery{Field: DeploymentName.String(), Value: "depname"},
								},
							},
						}},
					},
				}}},
			},
		}},
	}
	err := ValidateQuery(query, optionsMap)
	assert.Error(t, err)

	query = &v1.Query{
		Query: &v1.Query_Conjunction{Conjunction: &v1.ConjunctionQuery{
			Queries: []*v1.Query{
				{Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchFieldQuery{
							MatchFieldQuery: &v1.MatchFieldQuery{Field: CVE.String(), Value: "cveid"},
						},
					},
				}},
				{Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchFieldQuery{
							MatchFieldQuery: &v1.MatchFieldQuery{Field: CVSS.String(), Value: "2.0"},
						},
					},
				}},
			},
		}},
	}

	err = ValidateQuery(query, optionsMap)
	assert.NoError(t, err)
}
