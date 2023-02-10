//go:build sql_integration
// +build sql_integration

package postgres

import (
	"reflect"
	"strings"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search"
	mappings "github.com/stackrox/rox/pkg/search/options/deployments"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

var (
	deploymentBaseSchema = walker.Walk(reflect.TypeOf((*storage.Deployment)(nil)), "deployments")
)

func TestReplaceVars(t *testing.T) {
	cases := []struct {
		query  string
		result string
	}{
		{
			query:  "",
			result: "",
		},
		{
			"$$",
			"$1",
		},
		{
			query:  "select * from table where column > $$ and true",
			result: "select * from table where column > $1 and true",
		},
		{
			"$$ $$ $$ $$ $$ $$ $$ $$ $$ $$ $$",
			"$1 $2 $3 $4 $5 $6 $7 $8 $9 $10 $11",
		},
	}
	for _, c := range cases {
		t.Run(c.query, func(t *testing.T) {
			assert.Equal(t, c.result, replaceVars(c.query))
		})
	}
}

func BenchmarkReplaceVars(b *testing.B) {
	veryLongString := strings.Repeat("$$ ", 1000)
	b.Run("short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			replaceVars("$$ $$ $$ $$ $$ $$ $$ $$ $$ $$ $$")
		}
	})
	b.Run("long", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			replaceVars(veryLongString)
		}
	})
}

func TestMultiTableQueries(t *testing.T) {
	t.Parallel()

	deploymentBaseSchema.SetOptionsMap(mappings.OptionsMap)
	for _, c := range []struct {
		desc                 string
		q                    *v1.Query
		expectedQueryPortion string
		expectedFrom         string
		expectedWhere        string
		expectedData         []interface{}
		expectedJoinTables   []string
		expectedError        string
	}{
		{
			desc:          "base schema query",
			q:             search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			expectedFrom:  "deployments",
			expectedWhere: "deployments.Name = $$",
			expectedData:  []interface{}{"central"},
		},
		{
			desc:               "child schema query",
			q:                  search.NewQueryBuilder().AddExactMatches(search.ImageName, "stackrox").ProtoQuery(),
			expectedFrom:       "deployments",
			expectedWhere:      "deployments_containers.Image_Name_FullName = $$",
			expectedJoinTables: []string{"deployments_containers"},
			expectedData:       []interface{}{"stackrox"},
		},
		{
			desc: "base schema and child schema conjunction query",
			q: search.NewQueryBuilder().
				AddExactMatches(search.ImageName, "stackrox").
				AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			expectedFrom:       "deployments",
			expectedWhere:      "(deployments.Name = $$ and deployments_containers.Image_Name_FullName = $$)",
			expectedJoinTables: []string{"deployments_containers"},
			expectedData:       []interface{}{"central", "stackrox"},
		},
		{
			desc: "base schema and child schema disjunction query",
			q: search.DisjunctionQuery(
				search.NewQueryBuilder().AddExactMatches(search.ImageName, "stackrox").ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			),

			expectedFrom:       "deployments",
			expectedWhere:      "(deployments_containers.Image_Name_FullName = $$ or deployments.Name = $$)",
			expectedJoinTables: []string{"deployments_containers"},
			expectedData:       []interface{}{"central", "stackrox"},
		},
		{
			desc: "multiple child schema query",
			q: search.ConjunctionQuery(
				search.NewQueryBuilder().AddExactMatches(search.ImageName, "stackrox").ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.PortProtocol, "tcp").ProtoQuery(),
			),

			expectedFrom:       "deployments",
			expectedWhere:      "(deployments_containers.Image_Name_FullName = $$ and deployments_ports.Protocol = $$)",
			expectedJoinTables: []string{"deployments_containers", "deployments_ports"},
			expectedData:       []interface{}{"tcp", "stackrox"},
		},
		{
			desc: "multiple child schema disjunction query",
			q: search.DisjunctionQuery(
				search.NewQueryBuilder().AddExactMatches(search.ImageName, "stackrox").ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.PortProtocol, "tcp").ProtoQuery(),
			),
			expectedFrom:       "deployments",
			expectedWhere:      "(deployments_containers.Image_Name_FullName = $$ or deployments_ports.Protocol = $$)",
			expectedJoinTables: []string{"deployments_containers", "deployments_ports"},
			expectedData:       []interface{}{"tcp", "stackrox"},
		},
		{
			desc: "base schema and child schema disjunction query; bool+match",
			q: search.DisjunctionQuery(
				search.NewQueryBuilder().AddBools(search.Privileged, true).ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			),
			expectedFrom:       "deployments",
			expectedWhere:      "(deployments_containers.SecurityContext_Privileged = $$ or deployments.Name = $$)",
			expectedJoinTables: []string{"deployments_containers"},
			expectedData:       []interface{}{"central", "true"},
		},
		{
			desc:               "negated child schema query",
			q:                  search.NewQueryBuilder().AddStrings(search.ImageName, "!central").ProtoQuery(),
			expectedFrom:       "deployments",
			expectedWhere:      "NOT (deployments_containers.Image_Name_FullName ilike $$)",
			expectedJoinTables: []string{"deployments_containers"},
			expectedData:       []interface{}{"central%"},
		},
		{
			desc:          "nil query",
			q:             nil,
			expectedFrom:  "deployments",
			expectedWhere: "",
			expectedData:  []interface{}{},
		},
		{
			desc: "id and match non query",
			q: search.ConjunctionQuery(
				search.NewQueryBuilder().AddDocIDs("123").ProtoQuery(),
				search.MatchNoneQuery(),
			),
			expectedFrom:  "deployments",
			expectedWhere: "(deployments.Id = ANY($$::uuid[]) and false)",
			expectedData:  []interface{}{[]string{"123"}},
		},
		{
			desc: "base schema and child schema conjunction query on base ID",
			q: search.NewQueryBuilder().
				AddExactMatches(search.ImageName, "stackrox").
				AddExactMatches(search.DeploymentID, uuid.NewDummy().String()).ProtoQuery(),
			expectedFrom:       "deployments",
			expectedWhere:      "(deployments.Id = $$ and deployments_containers.Image_Name_FullName = $$)",
			expectedJoinTables: []string{"deployments_containers"},
			expectedData:       []interface{}{uuid.NewDummy(), "stackrox"},
		},
		{
			desc: "base schema and child schema conjunction query on base invalid ID",
			q: search.NewQueryBuilder().
				AddExactMatches(search.ImageName, "stackrox").
				AddExactMatches(search.DeploymentID, "not a uuid").ProtoQuery(),
			expectedError: `uuid: incorrect UUID length 10 in string "not a uuid"
        	            	value "not a uuid" in search query must be valid UUID`,
		},
	} {
		t.Run(c.desc, func(t *testing.T) {
			actual, err := standardizeQueryAndPopulatePath(c.q, deploymentBaseSchema, SEARCH)
			if c.expectedError != "" {
				assert.Error(t, err, c.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.expectedFrom, actual.From)
				assert.Equal(t, c.expectedWhere, actual.Where)
				assert.ElementsMatch(t, c.expectedData, actual.Data)
				var actualJoins []string
				for _, join := range actual.InnerJoins {
					actualJoins = append(actualJoins, join.rightTable)
				}
				assert.ElementsMatch(t, c.expectedJoinTables, actualJoins)
			}
		})
	}
}

func TestSelectQueries(t *testing.T) {
	t.Parallel()

	for _, c := range []struct {
		desc          string
		q             *v1.Query
		expectedError string
		expectedQuery string
	}{
		{
			desc: "base schema; no select",
			q: search.NewQueryBuilder().
				AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			expectedError: "select portion of the query cannot be empty",
		},
		{
			desc: "base schema; select",
			q: search.NewQueryBuilder().
				AddSelectFields(
					&v1.QueryField{
						Field: search.DeploymentName.String(),
					},
				).ProtoQuery(),
			expectedQuery: "select deployments.Name as deployment from deployments",
		},
		{
			desc: "base schema; select w/ where",
			q: search.NewQueryBuilder().
				AddSelectFields(
					&v1.QueryField{
						Field: search.DeploymentName.String(),
					},
				).
				AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			expectedQuery: "select deployments.Name as deployment from deployments where deployments.Name = $1",
		},
		{
			desc: "child schema; multiple select w/ where",
			q: search.NewQueryBuilder().
				AddSelectFields(
					&v1.QueryField{
						Field: search.Privileged.String(),
					},
					&v1.QueryField{
						Field: search.ImageName.String(),
					},
				).
				AddExactMatches(search.ImageName, "stackrox").ProtoQuery(),
			expectedQuery: "select deployments_containers.SecurityContext_Privileged as privileged, " +
				"deployments_containers.Image_Name_FullName as image " +
				"from deployments inner join deployments_containers " +
				"on deployments.Id = deployments_containers.deployments_Id " +
				"where deployments_containers.Image_Name_FullName = $1",
		},
		{
			desc: "child schema; multiple select w/ where & group by",
			q: search.NewQueryBuilder().
				AddSelectFields(
					&v1.QueryField{
						Field: search.Privileged.String(),
					},
					&v1.QueryField{
						Field: search.ImageName.String(),
					},
				).
				AddExactMatches(search.ImageName, "stackrox").
				AddGroupBy(search.Cluster, search.Namespace).ProtoQuery(),
			expectedQuery: "select jsonb_agg(deployments_containers.SecurityContext_Privileged) as privileged, " +
				"jsonb_agg(deployments_containers.Image_Name_FullName) as image, " +
				"deployments.ClusterName as cluster, deployments.Namespace as namespace " +
				"from deployments inner join deployments_containers " +
				"on deployments.Id = deployments_containers.deployments_Id " +
				"where deployments_containers.Image_Name_FullName = $1 " +
				"group by deployments.ClusterName, deployments.Namespace",
		},
		{
			desc: "base schema and child schema; select",
			q: search.NewQueryBuilder().
				AddSelectFields(
					&v1.QueryField{
						Field: search.DeploymentName.String(),
					},
					&v1.QueryField{
						Field: search.ImageName.String(),
					},
				).ProtoQuery(),
			expectedQuery: "select deployments.Name as deployment, deployments_containers.Image_Name_FullName as image " +
				"from deployments inner join deployments_containers on deployments.Id = deployments_containers.deployments_Id",
		},
		{
			desc: "base schema and child schema conjunction query; select w/ where",
			q: search.NewQueryBuilder().
				AddSelectFields(
					&v1.QueryField{
						Field: search.DeploymentName.String(),
					},
					&v1.QueryField{
						Field: search.ImageName.String(),
					},
				).
				AddExactMatches(search.ImageName, "stackrox").
				AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			expectedQuery: "select deployments.Name as deployment, deployments_containers.Image_Name_FullName as image " +
				"from deployments inner join deployments_containers " +
				"on deployments.Id = deployments_containers.deployments_Id " +
				"where (deployments.Name = $1 and deployments_containers.Image_Name_FullName = $2)",
		},
		{
			desc: "nil query",
			q:    nil,
		},
	} {
		t.Run(c.desc, func(t *testing.T) {
			actualQ, err := standardizeSelectQueryAndPopulatePath(c.q, schema.DeploymentsSchema, SELECT)
			if c.expectedError != "" {
				assert.Error(t, err, c.expectedError)
				return
			}
			if c.q == nil {
				assert.Nil(t, actualQ)
				return
			}

			actual := actualQ.AsSQL()
			assert.Equal(t, c.expectedQuery, actual)
		})
	}
}
