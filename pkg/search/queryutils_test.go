package search

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
)

func TestFilterQuery(t *testing.T) {
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
		}},
	}

	newQuery, filtered := FilterQueryWithMap(query, optionsMap)
	assert.True(t, filtered)
	protoassert.Equal(t, &v1.Query{
		Query: &v1.Query_BaseQuery{
			BaseQuery: &v1.BaseQuery{
				Query: &v1.BaseQuery_MatchFieldQuery{
					MatchFieldQuery: &v1.MatchFieldQuery{Field: CVE.String(), Value: "cveId"},
				},
			},
		},
	}, newQuery)

	var expected *v1.Query
	newQuery, filtered = FilterQueryWithMap(EmptyQuery(), optionsMap)
	assert.False(t, filtered)
	protoassert.Equal(t, expected, newQuery)

	q := &v1.Query{
		Query: &v1.Query_BaseQuery{
			BaseQuery: &v1.BaseQuery{
				Query: &v1.BaseQuery_MatchFieldQuery{
					MatchFieldQuery: &v1.MatchFieldQuery{
						Field: ImageSHA.String(),
						Value: "blah",
					},
				},
			},
		},
	}
	newQuery, filtered = FilterQueryWithMap(q, optionsMap)
	assert.False(t, filtered)
	protoassert.Equal(t, q, newQuery)
}

func TestSubQueryFieldScope(t *testing.T) {
	optionsMap := Walk(v1.SearchCategory_ADMINISTRATION_EVENTS, "event", &storage.AdministrationEvent{})
	inner := NewQueryBuilder().AddExactMatches(DeploymentName, "app").AddSelectFields(NewQuerySelect(ImageID)).ProtoQuery()
	query := MatchFieldSubQuery(ResourceID, v1.SearchCategory_DEPLOYMENTS, inner)
	var baseQueries []*v1.BaseQuery
	ApplyFnToAllBaseQueries(query, func(base *v1.BaseQuery) {
		baseQueries = append(baseQueries, base)
	})
	assert.Len(t, baseQueries, 1, "the subquery body belongs to another search scope")
	filtered, changed := FilterQueryWithMap(query, optionsMap)
	assert.False(t, changed)
	protoassert.Equal(t, query, filtered)
	inverse, changed := InverseFilterQueryWithMap(query, optionsMap)
	assert.False(t, changed)
	assert.Nil(t, inverse)
}

func TestInverseFilterQuery(t *testing.T) {
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
		}},
	}

	newQuery, filtered := InverseFilterQueryWithMap(query, optionsMap)
	assert.True(t, filtered)
	protoassert.Equal(t, &v1.Query{
		Query: &v1.Query_Conjunction{Conjunction: &v1.ConjunctionQuery{
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
		}},
	}, newQuery)

	var expected *v1.Query
	newQuery, filtered = InverseFilterQueryWithMap(EmptyQuery(), optionsMap)
	assert.False(t, filtered)
	protoassert.Equal(t, expected, newQuery)

	q := &v1.Query{
		Query: &v1.Query_BaseQuery{
			BaseQuery: &v1.BaseQuery{
				Query: &v1.BaseQuery_MatchFieldQuery{
					MatchFieldQuery: &v1.MatchFieldQuery{
						Field: ImageSHA.String(),
						Value: "blah",
					},
				},
			},
		},
	}
	newQuery, filtered = InverseFilterQueryWithMap(q, optionsMap)
	assert.False(t, filtered)
	protoassert.Equal(t, expected, newQuery)
}

func TestAddAsConjunction(t *testing.T) {
	toAdd := &v1.Query{
		Query: &v1.Query_BaseQuery{
			BaseQuery: &v1.BaseQuery{
				Query: &v1.BaseQuery_MatchFieldQuery{
					MatchFieldQuery: &v1.MatchFieldQuery{Field: CVE.String(), Value: "cveId"},
				},
			},
		},
	}

	addTo := &v1.Query{
		Query: &v1.Query_Conjunction{Conjunction: &v1.ConjunctionQuery{
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
		}},
	}

	expected := &v1.Query{
		Query: &v1.Query_Conjunction{Conjunction: &v1.ConjunctionQuery{
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
				{Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchFieldQuery{
							MatchFieldQuery: &v1.MatchFieldQuery{Field: CVE.String(), Value: "cveId"},
						},
					},
				}},
			},
		}},
	}

	added, err := AddAsConjunction(toAdd, addTo)
	assert.NoError(t, err)
	protoassert.Equal(t, expected, added)

	addTo = &v1.Query{
		Query: &v1.Query_BaseQuery{
			BaseQuery: &v1.BaseQuery{
				Query: &v1.BaseQuery_MatchFieldQuery{
					MatchFieldQuery: &v1.MatchFieldQuery{Field: DeploymentName.String(), Value: "depname"},
				},
			},
		},
	}

	expected = &v1.Query{
		Query: &v1.Query_Conjunction{Conjunction: &v1.ConjunctionQuery{
			Queries: []*v1.Query{
				{Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchFieldQuery{
							MatchFieldQuery: &v1.MatchFieldQuery{Field: CVE.String(), Value: "cveId"},
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
		}},
	}

	added, err = AddAsConjunction(toAdd, addTo)
	assert.NoError(t, err)
	protoassert.Equal(t, expected, added)

	addTo = &v1.Query{
		Query: &v1.Query_Disjunction{Disjunction: &v1.DisjunctionQuery{
			Queries: []*v1.Query{
				{Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchFieldQuery{
							MatchFieldQuery: &v1.MatchFieldQuery{Field: CVE.String(), Value: "cveId"},
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
		}},
	}

	_, err = AddAsConjunction(toAdd, addTo)
	assert.NoError(t, err)
}
