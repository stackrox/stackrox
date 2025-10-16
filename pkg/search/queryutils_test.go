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

	query := v1.Query_builder{
		Conjunction: v1.ConjunctionQuery_builder{
			Queries: []*v1.Query{
				v1.Query_builder{BaseQuery: v1.BaseQuery_builder{
					MatchFieldQuery: v1.MatchFieldQuery_builder{Field: CVE.String(), Value: "cveId"}.Build(),
				}.Build()}.Build(),
				v1.Query_builder{BaseQuery: v1.BaseQuery_builder{
					MatchFieldQuery: v1.MatchFieldQuery_builder{Field: DeploymentName.String(), Value: "depname"}.Build(),
				}.Build()}.Build(),
				v1.Query_builder{BaseQuery: v1.BaseQuery_builder{
					MatchFieldQuery: v1.MatchFieldQuery_builder{Field: DeploymentName.String(), Value: "depname"}.Build(),
				}.Build()}.Build(),
			},
		}.Build(),
	}.Build()

	newQuery, filtered := FilterQueryWithMap(query, optionsMap)
	assert.True(t, filtered)
	protoassert.Equal(t, v1.Query_builder{
		BaseQuery: v1.BaseQuery_builder{
			MatchFieldQuery: v1.MatchFieldQuery_builder{Field: CVE.String(), Value: "cveId"}.Build(),
		}.Build(),
	}.Build(), newQuery)

	var expected *v1.Query
	newQuery, filtered = FilterQueryWithMap(EmptyQuery(), optionsMap)
	assert.False(t, filtered)
	protoassert.Equal(t, expected, newQuery)

	q := v1.Query_builder{
		BaseQuery: v1.BaseQuery_builder{
			MatchFieldQuery: v1.MatchFieldQuery_builder{
				Field: ImageSHA.String(),
				Value: "blah",
			}.Build(),
		}.Build(),
	}.Build()
	newQuery, filtered = FilterQueryWithMap(q, optionsMap)
	assert.False(t, filtered)
	protoassert.Equal(t, q, newQuery)
}

func TestInverseFilterQuery(t *testing.T) {
	optionsMap := Walk(v1.SearchCategory_IMAGES, "derp", &storage.Image{})

	query := v1.Query_builder{
		Conjunction: v1.ConjunctionQuery_builder{
			Queries: []*v1.Query{
				v1.Query_builder{BaseQuery: v1.BaseQuery_builder{
					MatchFieldQuery: v1.MatchFieldQuery_builder{Field: CVE.String(), Value: "cveId"}.Build(),
				}.Build()}.Build(),
				v1.Query_builder{BaseQuery: v1.BaseQuery_builder{
					MatchFieldQuery: v1.MatchFieldQuery_builder{Field: DeploymentName.String(), Value: "depname"}.Build(),
				}.Build()}.Build(),
				v1.Query_builder{BaseQuery: v1.BaseQuery_builder{
					MatchFieldQuery: v1.MatchFieldQuery_builder{Field: DeploymentName.String(), Value: "depname"}.Build(),
				}.Build()}.Build(),
			},
		}.Build(),
	}.Build()

	newQuery, filtered := InverseFilterQueryWithMap(query, optionsMap)
	assert.True(t, filtered)
	protoassert.Equal(t, v1.Query_builder{
		Conjunction: v1.ConjunctionQuery_builder{
			Queries: []*v1.Query{
				v1.Query_builder{BaseQuery: v1.BaseQuery_builder{
					MatchFieldQuery: v1.MatchFieldQuery_builder{Field: DeploymentName.String(), Value: "depname"}.Build(),
				}.Build()}.Build(),
				v1.Query_builder{BaseQuery: v1.BaseQuery_builder{
					MatchFieldQuery: v1.MatchFieldQuery_builder{Field: DeploymentName.String(), Value: "depname"}.Build(),
				}.Build()}.Build(),
			},
		}.Build(),
	}.Build(), newQuery)

	var expected *v1.Query
	newQuery, filtered = InverseFilterQueryWithMap(EmptyQuery(), optionsMap)
	assert.False(t, filtered)
	protoassert.Equal(t, expected, newQuery)

	q := v1.Query_builder{
		BaseQuery: v1.BaseQuery_builder{
			MatchFieldQuery: v1.MatchFieldQuery_builder{
				Field: ImageSHA.String(),
				Value: "blah",
			}.Build(),
		}.Build(),
	}.Build()
	newQuery, filtered = InverseFilterQueryWithMap(q, optionsMap)
	assert.False(t, filtered)
	protoassert.Equal(t, expected, newQuery)
}

func TestAddAsConjunction(t *testing.T) {
	toAdd := v1.Query_builder{
		BaseQuery: v1.BaseQuery_builder{
			MatchFieldQuery: v1.MatchFieldQuery_builder{Field: CVE.String(), Value: "cveId"}.Build(),
		}.Build(),
	}.Build()

	addTo := v1.Query_builder{
		Conjunction: v1.ConjunctionQuery_builder{
			Queries: []*v1.Query{
				v1.Query_builder{BaseQuery: v1.BaseQuery_builder{
					MatchFieldQuery: v1.MatchFieldQuery_builder{Field: DeploymentName.String(), Value: "depname"}.Build(),
				}.Build()}.Build(),
				v1.Query_builder{BaseQuery: v1.BaseQuery_builder{
					MatchFieldQuery: v1.MatchFieldQuery_builder{Field: DeploymentName.String(), Value: "depname"}.Build(),
				}.Build()}.Build(),
			},
		}.Build(),
	}.Build()

	expected := v1.Query_builder{
		Conjunction: v1.ConjunctionQuery_builder{
			Queries: []*v1.Query{
				v1.Query_builder{BaseQuery: v1.BaseQuery_builder{
					MatchFieldQuery: v1.MatchFieldQuery_builder{Field: DeploymentName.String(), Value: "depname"}.Build(),
				}.Build()}.Build(),
				v1.Query_builder{BaseQuery: v1.BaseQuery_builder{
					MatchFieldQuery: v1.MatchFieldQuery_builder{Field: DeploymentName.String(), Value: "depname"}.Build(),
				}.Build()}.Build(),
				v1.Query_builder{BaseQuery: v1.BaseQuery_builder{
					MatchFieldQuery: v1.MatchFieldQuery_builder{Field: CVE.String(), Value: "cveId"}.Build(),
				}.Build()}.Build(),
			},
		}.Build(),
	}.Build()

	added, err := AddAsConjunction(toAdd, addTo)
	assert.NoError(t, err)
	protoassert.Equal(t, expected, added)

	addTo = v1.Query_builder{
		BaseQuery: v1.BaseQuery_builder{
			MatchFieldQuery: v1.MatchFieldQuery_builder{Field: DeploymentName.String(), Value: "depname"}.Build(),
		}.Build(),
	}.Build()

	expected = v1.Query_builder{
		Conjunction: v1.ConjunctionQuery_builder{
			Queries: []*v1.Query{
				v1.Query_builder{BaseQuery: v1.BaseQuery_builder{
					MatchFieldQuery: v1.MatchFieldQuery_builder{Field: CVE.String(), Value: "cveId"}.Build(),
				}.Build()}.Build(),
				v1.Query_builder{BaseQuery: v1.BaseQuery_builder{
					MatchFieldQuery: v1.MatchFieldQuery_builder{Field: DeploymentName.String(), Value: "depname"}.Build(),
				}.Build()}.Build(),
			},
		}.Build(),
	}.Build()

	added, err = AddAsConjunction(toAdd, addTo)
	assert.NoError(t, err)
	protoassert.Equal(t, expected, added)

	addTo = v1.Query_builder{
		Disjunction: v1.DisjunctionQuery_builder{
			Queries: []*v1.Query{
				v1.Query_builder{BaseQuery: v1.BaseQuery_builder{
					MatchFieldQuery: v1.MatchFieldQuery_builder{Field: CVE.String(), Value: "cveId"}.Build(),
				}.Build()}.Build(),
				v1.Query_builder{BaseQuery: v1.BaseQuery_builder{
					MatchFieldQuery: v1.MatchFieldQuery_builder{Field: DeploymentName.String(), Value: "depname"}.Build(),
				}.Build()}.Build(),
			},
		}.Build(),
	}.Build()

	_, err = AddAsConjunction(toAdd, addTo)
	assert.NoError(t, err)
}
