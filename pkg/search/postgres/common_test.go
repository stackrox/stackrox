package postgres

import (
	"reflect"
	"strings"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search"
	mappings "github.com/stackrox/rox/pkg/search/options/deployments"
	"github.com/stretchr/testify/assert"
)

var (
	deploymentBaseSchema = walker.Walk(reflect.TypeOf((*storage.Deployment)(nil)), "deployments")
)

func TestMultiTableQueries(t *testing.T) {
	t.Parallel()

	deploymentBaseSchema.SetOptionsMap(mappings.OptionsMap)
	for _, c := range []struct {
		desc     string
		q        *v1.Query
		expected *query
	}{
		{
			desc: "base schema query",
			q:    search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			expected: &query{
				Select: selectQuery{
					Query: "select deployments.Id",
				},
				From:  "deployments",
				Where: "(deployments.Name = $$)",
				Data:  []interface{}{"central"},
			},
		},
		{
			desc: "child schema query",
			q:    search.NewQueryBuilder().AddExactMatches(search.ImageName, "stackrox").ProtoQuery(),
			expected: &query{
				Select: selectQuery{
					Query: "select deployments.Id",
				},
				From:  "deployments, deployments_containers",
				Where: "(deployments_containers.Image_Name_FullName = $$ and deployments.Id = deployments_containers.deployments_Id)",
				Data:  []interface{}{"stackrox"},
			},
		},
		{
			desc: "base schema and child schema conjunction query",
			q: search.NewQueryBuilder().
				AddExactMatches(search.ImageName, "stackrox").
				AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			expected: &query{
				Select: selectQuery{
					Query: "select deployments.Id",
				},
				From:  "deployments, deployments_containers",
				Where: "((deployments.Name = $$) and (deployments_containers.Image_Name_FullName = $$ and deployments.Id = deployments_containers.deployments_Id))",
				Data:  []interface{}{"central", "stackrox"},
			},
		},
		{
			desc: "base schema and child schema disjunction query",
			q: search.DisjunctionQuery(
				search.NewQueryBuilder().AddExactMatches(search.ImageName, "stackrox").ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			),
			expected: &query{
				Select: selectQuery{
					Query: "select deployments.Id",
				},
				From:  "deployments, deployments_containers",
				Where: "((deployments_containers.Image_Name_FullName = $$ and deployments.Id = deployments_containers.deployments_Id) or (deployments.Name = $$))",
				Data:  []interface{}{"central", "stackrox"},
			},
		},
		{
			desc: "multiple child schema query",
			q: search.ConjunctionQuery(
				search.NewQueryBuilder().AddExactMatches(search.ImageName, "stackrox").ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.PortProtocol, "tcp").ProtoQuery(),
			),
			expected: &query{
				Select: selectQuery{
					Query: "select deployments.Id",
				},
				From:  "deployments, deployments_containers, deployments_ports",
				Where: "((deployments_containers.Image_Name_FullName = $$ and deployments.Id = deployments_containers.deployments_Id) and (deployments_ports.Protocol = $$ and deployments.Id = deployments_ports.deployments_Id))",
				Data:  []interface{}{"tcp", "stackrox"},
			},
		},
		{
			desc: "multiple child schema disjunction query",
			q: search.DisjunctionQuery(
				search.NewQueryBuilder().AddExactMatches(search.ImageName, "stackrox").ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.PortProtocol, "tcp").ProtoQuery(),
			),
			expected: &query{
				Select: selectQuery{
					Query: "select deployments.Id",
				},
				From:  "deployments, deployments_containers, deployments_ports",
				Where: "((deployments_containers.Image_Name_FullName = $$ and deployments.Id = deployments_containers.deployments_Id) or (deployments_ports.Protocol = $$ and deployments.Id = deployments_ports.deployments_Id))",
				Data:  []interface{}{"tcp", "stackrox"},
			},
		},
		{
			desc: "base schema and child schema disjunction query; bool+match",
			q: search.DisjunctionQuery(
				search.NewQueryBuilder().AddBools(search.Privileged, true).ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			),
			expected: &query{
				Select: selectQuery{
					Query: "select deployments.Id",
				},
				From:  "deployments, deployments_containers",
				Where: "((deployments_containers.SecurityContext_Privileged = $$ and deployments.Id = deployments_containers.deployments_Id) or (deployments.Name = $$))",
				Data:  []interface{}{"central", "true"},
			},
		},
		{
			desc: "negated child schema query",
			q:    search.NewQueryBuilder().AddStrings(search.ImageName, "!central").ProtoQuery(),
			expected: &query{
				Select: selectQuery{
					Query: "select deployments.Id",
				},
				From:  "deployments, deployments_containers",
				Where: "(NOT (deployments_containers.Image_Name_FullName ilike $$) and deployments.Id = deployments_containers.deployments_Id)",
				Data:  []interface{}{"central%"},
			},
		},
		{
			desc: "nil query",
			q:    nil,
			expected: &query{
				Select: selectQuery{
					Query: "select deployments.Id",
				},
				From:  "deployments",
				Where: "",
				Data:  []interface{}{},
			},
		},
		{
			desc: "id and match non query",
			q: search.ConjunctionQuery(
				search.NewQueryBuilder().AddDocIDs("123").ProtoQuery(),
				search.MatchNoneQuery(),
			),
			expected: &query{
				Select: selectQuery{
					Query: "select deployments.Id",
				},
				From:  "deployments",
				Where: "(deployments.Id = ANY($$::text[]) and false)",
				Data:  []interface{}{[]string{"123"}},
			},
		},
	} {
		t.Run(c.desc, func(t *testing.T) {
			actual, err := standardizeQueryAndPopulatePath(c.q, deploymentBaseSchema, GET)
			assert.NoError(t, err)
			assert.Equal(t, c.expected.Select, actual.Select)
			assert.ElementsMatch(t, strings.Split(c.expected.From, ", "), strings.Split(actual.From, ", "))
			assert.Equal(t, c.expected.Where, actual.Where)
			assert.ElementsMatch(t, c.expected.Data, actual.Data)
		})
	}
}
