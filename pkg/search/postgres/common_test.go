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

func TestGroupByString(t *testing.T) {
	t.Parallel()

	for _, c := range []struct {
		desc          string
		q             *v1.Query
		schema        *walker.Schema
		expectedQuery string
		expectedError bool
	}{
		{
			desc:          "group by primary key; having w/ derived field",
			q:             search.NewQueryBuilder().AddStrings(search.TestParent1ValCount, ">1").ProtoQuery(),
			schema:        schema.TestParent1Schema,
			expectedQuery: "select test_parent1.Id test_parent1_Id from test_parent1 group by test_parent1.Id having count(test_parent1.Val) > $1",
		},
		{
			desc: "group by non-primary key; having w/ derived field; no join",
			q: func() *v1.Query {
				q := search.NewQueryBuilder().AddStrings(search.TestParent1ValCount, ">2").ProtoQuery()
				q.GroupBy = &v1.QueryGroupBy{
					Fields: []string{search.TestParent1Val.String()},
				}
				return q
			}(),
			schema:        schema.TestParent1Schema,
			expectedQuery: "select unnest_t.f1, inner_t.Val from (select jsonb_agg(row(test_parent1.Id)), test_parent1.Val from test_parent1 group by test_parent1.Val having count(test_parent1.Val) > $1) inner_t, jsonb_to_recordset(inner_t.jsonb_agg) as unnest_t(f1 varchar)",
		},
		{
			desc: "group by non-primary key; having w/ derived field; w/ join",
			q: func() *v1.Query {
				q := search.NewQueryBuilder().AddStrings(search.TestChild1Count, ">5").ProtoQuery()
				q.GroupBy = &v1.QueryGroupBy{
					Fields: []string{search.TestGrandparentVal.String()},
				}
				return q
			}(),
			schema:        schema.TestGrandparentsSchema,
			expectedQuery: "select unnest_t.f1, inner_t.Val from (select jsonb_agg(row(test_grandparents.Id)), test_grandparents.Val from test_grandparents inner join test_parent1 on test_grandparents.Id = test_parent1.ParentId inner join test_parent1_childrens on test_parent1.Id = test_parent1_childrens.test_parent1_Id inner join test_child1 on test_parent1_childrens.ChildId = test_child1.Id group by test_grandparents.Val having count(test_child1.Id) > $1) inner_t, jsonb_to_recordset(inner_t.jsonb_agg) as unnest_t(f1 varchar)",
		},
		{
			desc: "group by derived field",
			q: func() *v1.Query {
				q := search.NewQueryBuilder().AddStrings(search.TestParent1ValCount, ">2").ProtoQuery()
				q.GroupBy = &v1.QueryGroupBy{
					Fields: []string{search.TestParent1ValCount.String()},
				}
				return q
			}(),
			schema:        schema.TestParent1Schema,
			expectedError: true,
		},
	} {
		t.Run(c.desc, func(t *testing.T) {
			actual, err := standardizeQueryAndPopulatePath(c.q, c.schema, SEARCH)
			if c.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.expectedQuery, actual.AsSQL())
			}
		})
	}
}
