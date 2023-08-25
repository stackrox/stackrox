package search

import (
	"fmt"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestParseRawQuery(t *testing.T) {
	testCases := []struct {
		desc          string
		queryStr      string
		shouldError   bool
		parser        generalQueryParser
		expectedQuery *v1.Query
	}{
		{
			desc:        "Query with ANDs and ORs",
			queryStr:    fmt.Sprintf("%s:field1,field12+%s:field2", DeploymentName, Category),
			shouldError: false,
			parser:      generalQueryParser{},
			expectedQuery: &v1.Query{
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
			},
		},
		{
			desc:        "Query with ANDs, ORs and extra white spaces",
			queryStr:    fmt.Sprintf("%s:field1,field12 + %s:field2", DeploymentName, Category),
			shouldError: false,
			parser:      generalQueryParser{},
			expectedQuery: &v1.Query{
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
			},
		},
		{
			desc:        "Empty query with MatchAllIfEmpty not set to true",
			queryStr:    "",
			shouldError: true,
			parser:      generalQueryParser{},
		},
		{
			desc:          "Empty query with MatchAllIfEmpty set to true",
			queryStr:      "",
			shouldError:   false,
			parser:        generalQueryParser{MatchAllIfEmpty: true},
			expectedQuery: EmptyQuery(),
		},
		{
			desc:        "Invalid query",
			queryStr:    "INVALIDQUERY",
			shouldError: true,
			parser:      generalQueryParser{},
		},
		{
			desc:        "Query with plus in double quotes",
			queryStr:    fmt.Sprintf("%s:field1,\"field12+some:thing\",field13 + %s:\"field2+something\"", DeploymentName, Category),
			shouldError: false,
			parser:      generalQueryParser{},
			expectedQuery: &v1.Query{
				Query: &v1.Query_Conjunction{Conjunction: &v1.ConjunctionQuery{
					Queries: []*v1.Query{
						{Query: &v1.Query_BaseQuery{
							BaseQuery: &v1.BaseQuery{
								Query: &v1.BaseQuery_MatchFieldQuery{
									MatchFieldQuery: &v1.MatchFieldQuery{Field: Category.String(), Value: "\"field2+something\""},
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
											MatchFieldQuery: &v1.MatchFieldQuery{Field: DeploymentName.String(), Value: "\"field12+some:thing\""},
										},
									},
								}},
								{Query: &v1.Query_BaseQuery{
									BaseQuery: &v1.BaseQuery{
										Query: &v1.BaseQuery_MatchFieldQuery{
											MatchFieldQuery: &v1.MatchFieldQuery{Field: DeploymentName.String(), Value: "field13"},
										},
									},
								}},
							},
						}}},
					},
				}},
			},
		},
		{
			desc:        "Query with plus and comma in double quotes",
			queryStr:    fmt.Sprintf("%s:field1,\"field12+some,thi:ng\",field13 + %s:\"field2+some,thing\"", DeploymentName, Category),
			shouldError: false,
			parser:      generalQueryParser{},
			expectedQuery: &v1.Query{
				Query: &v1.Query_Conjunction{Conjunction: &v1.ConjunctionQuery{
					Queries: []*v1.Query{
						{Query: &v1.Query_BaseQuery{
							BaseQuery: &v1.BaseQuery{
								Query: &v1.BaseQuery_MatchFieldQuery{
									MatchFieldQuery: &v1.MatchFieldQuery{Field: Category.String(), Value: "\"field2+some,thing\""},
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
											MatchFieldQuery: &v1.MatchFieldQuery{Field: DeploymentName.String(), Value: "\"field12+some,thi:ng\""},
										},
									},
								}},
								{Query: &v1.Query_BaseQuery{
									BaseQuery: &v1.BaseQuery{
										Query: &v1.BaseQuery_MatchFieldQuery{
											MatchFieldQuery: &v1.MatchFieldQuery{Field: DeploymentName.String(), Value: "field13"},
										},
									},
								}},
							},
						}}},
					},
				}},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			actualQuery, err := tc.parser.parse(tc.queryStr)
			if tc.shouldError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedQuery, actualQuery)
		})
	}
}
