package postgres

import (
	"context"
	"strings"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

var (
	deploymentBaseSchema   = schema.DeploymentsSchema
	imagesSchema           = schema.ImagesSchema
	imageCVEsSchema        = schema.ImageCvesSchema
	alertSchema            = schema.AlertsSchema
	_                      = schema.ImageCveEdgesSchema
	imageComponentV2Schema = schema.ImageComponentV2Schema
	imageCVEV2Schema       = schema.ImageCvesV2Schema
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

	for _, c := range []struct {
		desc                        string
		q                           *v1.Query
		schema                      *walker.Schema
		expectedQueryPortion        string
		expectedFrom                string
		expectedWhere               string
		expectedData                []interface{}
		expectedJoinTables          map[string]JoinType
		expectedError               string
		expectedFlattenedWhere      string
		expectedFlattenedJoinTables map[string]JoinType
	}{
		{
			desc:          "base schema query",
			q:             search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			schema:        deploymentBaseSchema,
			expectedFrom:  "deployments",
			expectedWhere: "deployments.Name = $$",
			expectedData:  []interface{}{"central"},
		},
		{
			desc:               "child schema query",
			q:                  search.NewQueryBuilder().AddExactMatches(search.ImageName, "stackrox").ProtoQuery(),
			schema:             deploymentBaseSchema,
			expectedFrom:       "deployments",
			expectedWhere:      "deployments_containers.Image_Name_FullName = $$",
			expectedJoinTables: map[string]JoinType{"deployments_containers": Inner},
			expectedData:       []interface{}{"stackrox"},
		},
		{
			desc: "base schema and child schema conjunction query",
			q: search.NewQueryBuilder().
				AddExactMatches(search.ImageName, "stackrox").
				AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			schema:             deploymentBaseSchema,
			expectedFrom:       "deployments",
			expectedWhere:      "(deployments.Name = $$ and deployments_containers.Image_Name_FullName = $$)",
			expectedJoinTables: map[string]JoinType{"deployments_containers": Inner},
			expectedData:       []interface{}{"central", "stackrox"},
		},
		{
			desc: "base schema and child schema disjunction query",
			q: search.DisjunctionQuery(
				search.NewQueryBuilder().AddExactMatches(search.ImageName, "stackrox").ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			),
			schema:             deploymentBaseSchema,
			expectedFrom:       "deployments",
			expectedWhere:      "(deployments_containers.Image_Name_FullName = $$ or deployments.Name = $$)",
			expectedJoinTables: map[string]JoinType{"deployments_containers": Inner},
			expectedData:       []interface{}{"central", "stackrox"},
		},
		{
			desc: "multiple child schema query",
			q: search.ConjunctionQuery(
				search.NewQueryBuilder().AddExactMatches(search.ImageName, "stackrox").ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.PortProtocol, "tcp").ProtoQuery(),
			),
			schema:             deploymentBaseSchema,
			expectedFrom:       "deployments",
			expectedWhere:      "(deployments_containers.Image_Name_FullName = $$ and deployments_ports.Protocol = $$)",
			expectedJoinTables: map[string]JoinType{"deployments_containers": Inner, "deployments_ports": Inner},
			expectedData:       []interface{}{"tcp", "stackrox"},
		},
		{
			desc: "multiple child schema disjunction query",
			q: search.DisjunctionQuery(
				search.NewQueryBuilder().AddExactMatches(search.ImageName, "stackrox").ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.PortProtocol, "tcp").ProtoQuery(),
			),
			schema:             deploymentBaseSchema,
			expectedFrom:       "deployments",
			expectedWhere:      "(deployments_containers.Image_Name_FullName = $$ or deployments_ports.Protocol = $$)",
			expectedJoinTables: map[string]JoinType{"deployments_containers": Inner, "deployments_ports": Inner},
			expectedData:       []interface{}{"tcp", "stackrox"},
		},
		{
			desc: "base schema and child schema disjunction query; bool+match",
			q: search.DisjunctionQuery(
				search.NewQueryBuilder().AddBools(search.Privileged, true).ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			),
			schema:             deploymentBaseSchema,
			expectedFrom:       "deployments",
			expectedWhere:      "(deployments_containers.SecurityContext_Privileged = $$ or deployments.Name = $$)",
			expectedJoinTables: map[string]JoinType{"deployments_containers": Inner},
			expectedData:       []interface{}{"central", "true"},
		},
		{
			desc:               "negated child schema query",
			q:                  search.NewQueryBuilder().AddStrings(search.ImageName, "!central").ProtoQuery(),
			schema:             deploymentBaseSchema,
			expectedFrom:       "deployments",
			expectedWhere:      "NOT (deployments_containers.Image_Name_FullName ilike $$)",
			expectedJoinTables: map[string]JoinType{"deployments_containers": Inner},
			expectedData:       []interface{}{"central%"},
		},
		{
			desc:          "nil query",
			q:             nil,
			schema:        deploymentBaseSchema,
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
			schema:        deploymentBaseSchema,
			expectedFrom:  "deployments",
			expectedWhere: "(deployments.Id = ANY($$::uuid[]) and false)",
			expectedData:  []interface{}{[]string{"123"}},
		},
		{
			desc: "base schema and child schema conjunction query on base ID",
			q: search.NewQueryBuilder().
				AddExactMatches(search.ImageName, "stackrox").
				AddExactMatches(search.DeploymentID, uuid.NewDummy().String()).ProtoQuery(),
			schema:             deploymentBaseSchema,
			expectedFrom:       "deployments",
			expectedWhere:      "(deployments.Id = $$ and deployments_containers.Image_Name_FullName = $$)",
			expectedJoinTables: map[string]JoinType{"deployments_containers": Inner},
			expectedData:       []interface{}{uuid.NewDummy(), "stackrox"},
		},
		{
			desc: "base schema and child schema conjunction query on base invalid ID",
			q: search.NewQueryBuilder().
				AddExactMatches(search.ImageName, "stackrox").
				AddExactMatches(search.DeploymentID, "not a uuid").ProtoQuery(),
			schema: deploymentBaseSchema,
			expectedError: `uuid: incorrect UUID length 10 in string "not a uuid"
        	            	value "not a uuid" in search query must be valid UUID`,
		},
		{
			desc: "search of child schema mutliple results for base ID",
			q: search.NewQueryBuilder().AddLinkedFieldsHighlighted(
				[]search.FieldLabel{search.ImageName, search.EnvironmentKey},
				[]string{search.WildcardString, search.WildcardString}).
				ProtoQuery(),
			schema:             deploymentBaseSchema,
			expectedFrom:       "deployments",
			expectedWhere:      "(deployments_containers.Image_Name_FullName is not null and deployments_containers_envs.Key is not null)",
			expectedJoinTables: map[string]JoinType{"deployments_containers": Inner, "deployments_containers_envs": Inner},
		},
		{
			desc: "search active and inactive images with observed CVEs in non-platform deployments",
			q: search.NewQueryBuilder().
				AddExactMatches(search.VulnerabilityState, storage.VulnerabilityState_OBSERVED.String()).
				AddStrings(search.PlatformComponent, "false", "-").
				ProtoQuery(),
			schema:        imagesSchema,
			expectedFrom:  "images",
			expectedWhere: "((deployments.PlatformComponent = $$ or deployments.PlatformComponent is null) and image_cve_edges.State = $$)",
			expectedJoinTables: map[string]JoinType{
				"image_component_edges":     Inner,
				"image_component_cve_edges": Inner,
				"image_cves":                Inner,
				"image_cve_edges":           Inner,
				"deployments_containers":    Left,
				"deployments":               Left,
			},
			expectedData:           []interface{}{"false", "0"},
			expectedFlattenedWhere: "((deployments.PlatformComponent = $$ or deployments.PlatformComponent is null) and image_cves_v2.State = $$)",
			expectedFlattenedJoinTables: map[string]JoinType{
				"image_cves_v2":          Inner,
				"deployments_containers": Left,
				"deployments":            Left,
			},
		},
	} {
		t.Run(c.desc, func(t *testing.T) {
			ctx := sac.WithAllAccess(context.Background())
			actual, err := standardizeQueryAndPopulatePath(ctx, c.q, c.schema, SEARCH)
			if c.expectedError != "" {
				assert.Error(t, err, c.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.expectedFrom, actual.From)
				expectedWhere := c.expectedWhere
				if features.FlattenCVEData.Enabled() && c.expectedFlattenedWhere != "" {
					expectedWhere = c.expectedFlattenedWhere
				}
				assert.Equal(t, expectedWhere, actual.Where)
				assert.ElementsMatch(t, c.expectedData, actual.Data)

				var actualJoins map[string]JoinType
				if len(actual.Joins) > 0 {
					actualJoins = make(map[string]JoinType)
					for _, join := range actual.Joins {
						actualJoins[join.rightTable] = join.joinType
					}
				}
				expectedJoinTables := c.expectedJoinTables
				if features.FlattenCVEData.Enabled() && c.expectedFlattenedJoinTables != nil {
					expectedJoinTables = c.expectedFlattenedJoinTables
				}
				assert.Equal(t, expectedJoinTables, actualJoins)
			}
		})
	}
}

func TestCountQueries(t *testing.T) {
	baseCtx := sac.WithAllAccess(context.Background())
	for _, c := range []struct {
		desc                       string
		ctx                        context.Context
		q                          *v1.Query
		schema                     *walker.Schema
		expectedStatement          string
		expectedData               []interface{}
		expectedError              string
		expectedFlattenedStatement string
	}{
		{
			desc:              "base schema query",
			ctx:               baseCtx,
			q:                 search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			schema:            deploymentBaseSchema,
			expectedStatement: normalizeStatement(`select count(*) from deployments where deployments.Name = $1`),
			expectedData:      []interface{}{"central"},
		},
		{
			desc:              "base schema query, multiple matches",
			ctx:               baseCtx,
			q:                 search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "A", "B").ProtoQuery(),
			schema:            deploymentBaseSchema,
			expectedStatement: "select count(*) from deployments where deployments.Name IN ($1, $2)",
			expectedData:      []interface{}{"A", "B"},
		},
		{
			desc:   "child schema query",
			ctx:    baseCtx,
			q:      search.NewQueryBuilder().AddExactMatches(search.ImageName, "stackrox").ProtoQuery(),
			schema: deploymentBaseSchema,
			expectedStatement: normalizeStatement(`select count(distinct(deployments.Id)) from deployments
				inner join deployments_containers on deployments.Id = deployments_containers.deployments_Id
				where deployments_containers.Image_Name_FullName = $1`),
			expectedData: []interface{}{"stackrox"},
		},
		{
			desc: "base schema and child schema conjunction query",
			ctx:  baseCtx,
			q: search.NewQueryBuilder().
				AddExactMatches(search.ImageName, "stackrox").
				AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			schema: deploymentBaseSchema,
			expectedStatement: normalizeStatement(`select count(distinct(deployments.Id)) from deployments
				inner join deployments_containers on deployments.Id = deployments_containers.deployments_Id
				where (deployments.Name = $1 and deployments_containers.Image_Name_FullName = $2)`),
			expectedData: []interface{}{"central", "stackrox"},
		},
		{
			desc: "base schema and child schema disjunction query",
			ctx:  baseCtx,
			q: search.DisjunctionQuery(
				search.NewQueryBuilder().AddExactMatches(search.ImageName, "stackrox").ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			),
			schema: deploymentBaseSchema,
			expectedStatement: normalizeStatement(`select count(distinct(deployments.Id)) from deployments
				inner join deployments_containers on deployments.Id = deployments_containers.deployments_Id
				where (deployments_containers.Image_Name_FullName = $1 or deployments.Name = $2)`),
			expectedData: []interface{}{"stackrox", "central"},
		},
		{
			desc: "base schema and child schema disjunction query multiple values",
			ctx:  baseCtx,
			q: search.DisjunctionQuery(
				search.NewQueryBuilder().AddExactMatches(search.ImageName, "A", "B").ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "X", "Y", "Z").ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "X").ProtoQuery(),
			),
			schema: deploymentBaseSchema,
			expectedStatement: normalizeStatement(`select count(distinct(deployments.Id)) from deployments
				inner join deployments_containers on deployments.Id = deployments_containers.deployments_Id
				where (deployments_containers.Image_Name_FullName IN ($1, $2) or deployments.Name IN ($3, $4, $5) or deployments.Name = $6)`),
			expectedData: []interface{}{"A", "B", "X", "Y", "Z", "X"},
		},
		{
			desc: "base schema mix of conjunction and disjunction queries",
			ctx:  baseCtx,
			q: search.ConjunctionQuery(
				search.ConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ClusterID, fixtureconsts.Cluster1).ProtoQuery(),
					search.NewQueryBuilder().AddExactMatches(search.Namespace, "namespace").ProtoQuery(),
				),
				search.NewQueryBuilder().
					AddExactMatches(
						search.ViolationState,
						storage.ViolationState_ATTEMPTED.String(),
						storage.ViolationState_ACTIVE.String(),
					).
					ProtoQuery(),
			),
			schema: alertSchema,
			expectedStatement: normalizeStatement(`select count(*) from alerts
				where ((alerts.ClusterId = $1 and alerts.Namespace = $2) and alerts.State IN ($3, $4))`),
			expectedData: []interface{}{uuid.FromStringOrPanic("caaaaaaa-bbbb-4011-0000-111111111111"), "namespace", "3", "0"},
		},
		{
			desc: "multiple child schema query",
			ctx:  baseCtx,
			q: search.ConjunctionQuery(
				search.NewQueryBuilder().AddExactMatches(search.ImageName, "stackrox").ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.PortProtocol, "tcp").ProtoQuery(),
			),
			schema: deploymentBaseSchema,
			expectedStatement: normalizeStatement(`select count(distinct(deployments.Id)) from deployments
				inner join deployments_containers on deployments.Id = deployments_containers.deployments_Id
				inner join deployments_ports on deployments.Id = deployments_ports.deployments_Id
				where (deployments_containers.Image_Name_FullName = $1 and deployments_ports.Protocol = $2)`),
			expectedData: []interface{}{"stackrox", "tcp"},
		},
		{
			desc: "multiple child schema disjunction query",
			ctx:  baseCtx,
			q: search.DisjunctionQuery(
				search.NewQueryBuilder().AddExactMatches(search.ImageName, "stackrox").ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.PortProtocol, "tcp").ProtoQuery(),
			),
			schema: deploymentBaseSchema,
			expectedStatement: normalizeStatement(`select count(distinct(deployments.Id)) from deployments
				inner join deployments_containers on deployments.Id = deployments_containers.deployments_Id
				inner join deployments_ports on deployments.Id = deployments_ports.deployments_Id
				where (deployments_containers.Image_Name_FullName = $1 or deployments_ports.Protocol = $2)`),
			expectedData: []interface{}{"stackrox", "tcp"},
		},
		{
			desc: "base schema and child schema disjunction query; bool+match",
			ctx:  baseCtx,
			q: search.DisjunctionQuery(
				search.NewQueryBuilder().AddBools(search.Privileged, true).ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			),
			schema: deploymentBaseSchema,
			expectedStatement: normalizeStatement(`select count(distinct(deployments.Id)) from deployments
				inner join deployments_containers on deployments.Id = deployments_containers.deployments_Id
				where (deployments_containers.SecurityContext_Privileged = $1 or deployments.Name = $2)`),
			expectedData: []interface{}{"true", "central"},
		},
		{
			desc:   "negated child schema query",
			ctx:    baseCtx,
			q:      search.NewQueryBuilder().AddStrings(search.ImageName, "!central").ProtoQuery(),
			schema: deploymentBaseSchema,
			expectedStatement: normalizeStatement(`select count(distinct(deployments.Id)) from deployments
				inner join deployments_containers on deployments.Id = deployments_containers.deployments_Id
				where NOT (deployments_containers.Image_Name_FullName ilike $1)`),
			expectedData: []interface{}{"central%"},
		},
		{
			desc:              "nil query",
			ctx:               baseCtx,
			q:                 nil,
			schema:            deploymentBaseSchema,
			expectedStatement: normalizeStatement(`select count(*) from deployments`),
			expectedData:      []interface{}(nil),
		},
		{
			desc: "id and match non query",
			ctx:  baseCtx,
			q: search.ConjunctionQuery(
				search.NewQueryBuilder().AddDocIDs("123").ProtoQuery(),
				search.MatchNoneQuery(),
			),
			schema: deploymentBaseSchema,
			expectedStatement: normalizeStatement(`select count(*) from deployments
				where (deployments.Id = ANY($1::uuid[]) and false)`),
			expectedData: []interface{}{[]string{"123"}},
		},
		{
			desc: "base schema and child schema conjunction query on base ID",
			ctx:  baseCtx,
			q: search.NewQueryBuilder().
				AddExactMatches(search.ImageName, "stackrox").
				AddExactMatches(search.DeploymentID, uuid.NewDummy().String()).ProtoQuery(),
			schema: deploymentBaseSchema,
			expectedStatement: normalizeStatement(`select count(distinct(deployments.Id)) from deployments
				inner join deployments_containers on deployments.Id = deployments_containers.deployments_Id
				where (deployments.Id = $1 and deployments_containers.Image_Name_FullName = $2)`),
			expectedData: []interface{}{uuid.NewDummy(), "stackrox"},
		},
		{
			desc: "base schema and child schema conjunction query on base invalid ID",
			ctx:  baseCtx,
			q: search.NewQueryBuilder().
				AddExactMatches(search.ImageName, "stackrox").
				AddExactMatches(search.DeploymentID, "not a uuid").ProtoQuery(),
			schema: deploymentBaseSchema,
			expectedError: `uuid: incorrect UUID length 10 in string "not a uuid"
				value "not a uuid" in search query must be valid UUID`,
		},
		{
			desc: "search of child schema mutliple results for base ID",
			ctx:  baseCtx,
			q: search.NewQueryBuilder().AddLinkedFieldsHighlighted(
				[]search.FieldLabel{search.ImageName, search.EnvironmentKey},
				[]string{search.WildcardString, search.WildcardString}).
				ProtoQuery(),
			schema: deploymentBaseSchema,
			expectedStatement: normalizeStatement(`select count(distinct(deployments.Id)) from deployments
				inner join deployments_containers on deployments.Id = deployments_containers.deployments_Id
				inner join deployments_containers_envs on deployments_containers.deployments_Id = deployments_containers_envs.deployments_Id
				and deployments_containers.idx = deployments_containers_envs.deployments_containers_idx
				where (deployments_containers.Image_Name_FullName is not null and deployments_containers_envs.Key is not null)`),
		},
		{
			desc: "search active and inactive images with observed CVEs in non-platform deployments",
			ctx:  baseCtx,
			q: search.NewQueryBuilder().
				AddExactMatches(search.VulnerabilityState, storage.VulnerabilityState_OBSERVED.String()).
				AddStrings(search.PlatformComponent, "false", "-").
				ProtoQuery(),
			schema: imagesSchema,
			expectedStatement: normalizeStatement(`select count(distinct(images.Id)) from images
				left join deployments_containers on images.Id = deployments_containers.Image_Id
				left join deployments on deployments_containers.deployments_Id = deployments.Id
				inner join image_component_edges on images.Id = image_component_edges.ImageId
				inner join image_component_cve_edges on image_component_edges.ImageComponentId = image_component_cve_edges.ImageComponentId
				inner join image_cves on image_component_cve_edges.ImageCveId = image_cves.Id
				inner join image_cve_edges on(images.Id = image_cve_edges.ImageId and image_component_cve_edges.ImageCveId = image_cve_edges.ImageCveId)
				where ((deployments.PlatformComponent = $1 or deployments.PlatformComponent is null) and image_cve_edges.State = $2)`),
			expectedData: []interface{}{"false", "0"},
			expectedFlattenedStatement: `select count(distinct(images.Id)) from images
				left join deployments_containers on images.Id = deployments_containers.Image_Id
				left join deployments on deployments_containers.deployments_Id = deployments.Id
				inner join image_cves_v2 on images.Id = image_cves_v2.ImageId
				where ((deployments.PlatformComponent = $1 or deployments.PlatformComponent is null) and image_cves_v2.State = $2)`,
		},
	} {
		t.Run(c.desc, func(it *testing.T) {
			actual, err := standardizeQueryAndPopulatePath(c.ctx, c.q, c.schema, COUNT)
			if c.expectedError != "" {
				assert.Error(it, err, c.expectedError)
			} else {
				assert.NoError(it, err)
				expectedStatement := c.expectedStatement
				if features.FlattenCVEData.Enabled() && c.expectedFlattenedStatement != "" {
					expectedStatement = c.expectedFlattenedStatement
				}
				assert.Equal(it, expectedStatement, actual.AsSQL())
				assert.Equal(it, c.expectedData, actual.Data)
			}
		})
	}
}

func TestSelectQueries(t *testing.T) {
	t.Parallel()

	for _, c := range []struct {
		desc                   string
		ctx                    context.Context
		q                      *v1.Query
		schema                 *walker.Schema
		flattenedSchema        *walker.Schema
		expectedError          string
		expectedQuery          string
		expectedFlattenedQuery string
	}{
		{
			desc: "base schema; no select",
			q: search.NewQueryBuilder().
				AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			schema:        deploymentBaseSchema,
			expectedError: "select portion of the query cannot be empty",
		},
		{
			desc: "base schema; select",
			q: search.NewQueryBuilder().
				AddSelectFields(search.NewQuerySelect(search.DeploymentName)).ProtoQuery(),
			schema:        deploymentBaseSchema,
			expectedQuery: "select deployments.Name as deployment from deployments",
		},
		{
			desc: "base schema; select w/ where",
			q: search.NewQueryBuilder().
				AddSelectFields(search.NewQuerySelect(search.DeploymentName)).
				AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			schema:        deploymentBaseSchema,
			expectedQuery: "select deployments.Name as deployment from deployments where deployments.Name = $1",
		},
		{
			desc: "base schema; select w/ regexes",
			q: search.NewQueryBuilder().
				AddSelectFields(search.NewQuerySelect(search.DeploymentName)).
				AddRegexes(search.DeploymentName, "A", "B").ProtoQuery(),
			schema:        deploymentBaseSchema,
			expectedQuery: "select deployments.Name as deployment from deployments where (deployments.Name ~* $1 or deployments.Name ~* $2)",
		},
		{
			desc: "child schema; multiple select w/ where",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.Privileged),
					search.NewQuerySelect(search.ImageName),
				).
				AddExactMatches(search.ImageName, "stackrox").ProtoQuery(),
			schema: deploymentBaseSchema,
			expectedQuery: normalizeStatement(`select deployments_containers.SecurityContext_Privileged as privileged,
				deployments_containers.Image_Name_FullName as image
				from deployments inner join deployments_containers
				on deployments.Id = deployments_containers.deployments_Id
				where deployments_containers.Image_Name_FullName = $1`),
		},
		{
			desc: "child schema; multiple values highlighted (IN)",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.Privileged),
					search.NewQuerySelect(search.ImageName),
				).
				AddExactMatches(search.ImageName, "A", "B").MarkHighlighted(search.ImageName).ProtoQuery(),
			schema: deploymentBaseSchema,
			expectedQuery: normalizeStatement(`select deployments_containers.SecurityContext_Privileged as privileged,
				deployments_containers.Image_Name_FullName as image,
				deployments_containers.Image_Name_FullName
				from deployments inner join deployments_containers
				on deployments.Id = deployments_containers.deployments_Id
				where deployments_containers.Image_Name_FullName IN ($1, $2)`),
		},
		{
			desc: "child schema; multiple select w/ where & group by",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.Privileged),
					search.NewQuerySelect(search.ImageName),
				).
				AddExactMatches(search.ImageName, "stackrox").
				AddGroupBy(search.Cluster, search.Namespace).ProtoQuery(),
			schema: deploymentBaseSchema,
			expectedQuery: normalizeStatement(`select jsonb_agg(deployments_containers.SecurityContext_Privileged) as privileged,
				jsonb_agg(deployments_containers.Image_Name_FullName) as image,
				deployments.ClusterName as cluster, deployments.Namespace as namespace
				from deployments inner join deployments_containers
				on deployments.Id = deployments_containers.deployments_Id
				where deployments_containers.Image_Name_FullName = $1
				group by deployments.ClusterName, deployments.Namespace`),
		},
		{
			desc: "child schema; multiple select w/ where & group by with IN",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.Privileged),
					search.NewQuerySelect(search.ImageName),
				).
				AddExactMatches(search.ImageName, "stack", "rox").
				AddGroupBy(search.Cluster, search.Namespace).ProtoQuery(),
			schema: deploymentBaseSchema,
			expectedQuery: normalizeStatement(`select jsonb_agg(deployments_containers.SecurityContext_Privileged) as privileged,
				jsonb_agg(deployments_containers.Image_Name_FullName) as image,
				deployments.ClusterName as cluster, deployments.Namespace as namespace
				from deployments inner join deployments_containers
				on deployments.Id = deployments_containers.deployments_Id
				where deployments_containers.Image_Name_FullName IN ($1, $2)
				group by deployments.ClusterName, deployments.Namespace`),
		},
		{
			desc: "base schema and child schema; select",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.DeploymentName),
					search.NewQuerySelect(search.ImageName),
				).ProtoQuery(),
			schema: deploymentBaseSchema,
			expectedQuery: normalizeStatement(`select deployments.Name as deployment, deployments_containers.Image_Name_FullName as image
				from deployments inner join deployments_containers on deployments.Id = deployments_containers.deployments_Id`),
		},
		{
			desc: "base schema and child schema conjunction query; select w/ where",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.DeploymentName),
					search.NewQuerySelect(search.ImageName),
				).
				AddExactMatches(search.ImageName, "stackrox").
				AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			schema: deploymentBaseSchema,
			expectedQuery: normalizeStatement(`select deployments.Name as deployment, deployments_containers.Image_Name_FullName as image
				from deployments inner join deployments_containers
				on deployments.Id = deployments_containers.deployments_Id
				where (deployments.Name = $1 and deployments_containers.Image_Name_FullName = $2)`),
		},
		{
			desc: "derived field select",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.DeploymentName).AggrFunc(aggregatefunc.Count).Distinct(),
				).ProtoQuery(),
			schema:        deploymentBaseSchema,
			expectedQuery: "select count(distinct(deployments.Name)) as deployment_count from deployments",
		},
		{
			desc: "derived field select w/ where",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.DeploymentName).AggrFunc(aggregatefunc.Count).Distinct(),
				).
				AddExactMatches(search.ImageName, "stackrox").
				AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			schema: deploymentBaseSchema,
			expectedQuery: normalizeStatement(`select count(distinct(deployments.Name)) as deployment_count
				from deployments inner join deployments_containers
				on deployments.Id = deployments_containers.deployments_Id
				where (deployments.Name = $1 and deployments_containers.Image_Name_FullName = $2)`),
		},
		{
			desc:   "nil query",
			q:      nil,
			schema: deploymentBaseSchema,
		},
		{
			desc: "base schema; select w/ conjunction",
			q: func() *v1.Query {
				q := search.ConjunctionQuery(
					search.NewQueryBuilder().
						AddExactMatches(search.DeploymentName, "dep").ProtoQuery(),
					search.NewQueryBuilder().
						AddExactMatches(search.Namespace, "ns").ProtoQuery(),
				)
				q.Selects = []*v1.QuerySelect{search.NewQuerySelect(search.DeploymentName).Proto()}
				return q
			}(),
			schema:        deploymentBaseSchema,
			expectedQuery: "select deployments.Name as deployment from deployments where (deployments.Name = $1 and deployments.Namespace = $2)",
		},
		{
			desc: "base schema; select w/ where; image scope",
			ctx: scoped.Context(context.Background(), scoped.Scope{
				ID:    "fake-image",
				Level: v1.SearchCategory_IMAGES,
			}),
			q: search.NewQueryBuilder().
				AddSelectFields(search.NewQuerySelect(search.DeploymentName)).
				AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			schema: deploymentBaseSchema,
			expectedQuery: normalizeStatement(`select deployments.Name as deployment from deployments
				inner join deployments_containers on deployments.Id = deployments_containers.deployments_Id
				where (deployments.Name = $1 and deployments_containers.Image_Id = $2)`),
		},
		{
			desc: "base schema; select w/ multiple scopes",
			ctx: scoped.Context(context.Background(), scoped.Scope{
				ID:    uuid.NewV4().String(),
				Level: v1.SearchCategory_NAMESPACES,
				Parent: &scoped.Scope{
					ID:    uuid.NewV4().String(),
					Level: v1.SearchCategory_CLUSTERS,
				},
			}),
			q: search.NewQueryBuilder().
				AddSelectFields(search.NewQuerySelect(search.DeploymentName)).
				AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			schema: deploymentBaseSchema,
			expectedQuery: normalizeStatement(`select deployments.Name as deployment from deployments
				where (deployments.Name = $1 and (deployments.NamespaceId = $2 and deployments.ClusterId = $3))`),
		},
		{
			desc: "select query with filters that will add left joins to the query",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.CVE),
					search.NewQuerySelect(search.CVEID).Distinct(),
					search.NewQuerySelect(search.CVSS).AggrFunc(aggregatefunc.Max),
					search.NewQuerySelect(search.ImageSHA).AggrFunc(aggregatefunc.Count).Distinct(),
				).
				AddExactMatches(search.VulnerabilityState, storage.VulnerabilityState_OBSERVED.String()).
				AddStrings(search.PlatformComponent, "true", "-").
				ProtoQuery(),
			schema:          imageCVEsSchema,
			flattenedSchema: imageComponentV2Schema,
			expectedQuery: normalizeStatement(`select image_cves.CveBaseInfo_Cve as cve,
				distinct(image_cves.Id) as cve_id, max(image_cves.Cvss) as cvss_max,
				count(distinct(images.Id)) as image_sha_count
				from image_cves
				inner join image_component_cve_edges on image_cves.Id = image_component_cve_edges.ImageCveId
				inner join image_component_edges on image_component_cve_edges.ImageComponentId = image_component_edges.ImageComponentId
				inner join images on image_component_edges.ImageId = images.Id left join deployments_containers on images.Id = deployments_containers.Image_Id
				left join deployments on deployments_containers.deployments_Id = deployments.Id
				inner join image_cve_edges on(image_component_edges.ImageId = image_cve_edges.ImageId and image_cves.Id = image_cve_edges.ImageCveId)
				where ((deployments.PlatformComponent = $1 or deployments.PlatformComponent is null) and image_cve_edges.State = $2)`),
			expectedFlattenedQuery: "select image_cves_v2.CveBaseInfo_Cve as cve, " +
				"distinct(image_cves_v2.Id) as cve_id, max(image_cves_v2.Cvss) as cvss_max, " +
				"count(distinct(images.Id)) as image_sha_count " +
				"from image_component_v2 " +
				"inner join image_cves_v2 on image_component_v2.Id = image_cves_v2.ComponentId " +
				"inner join images on image_component_v2.ImageId = images.Id " +
				"left join deployments_containers on images.Id = deployments_containers.Image_Id " +
				"left join deployments on deployments_containers.deployments_Id = deployments.Id " +
				"where ((deployments.PlatformComponent = $1 or deployments.PlatformComponent is null) and image_cves_v2.State = $2)",
		},
		{
			desc: "select with multiple enum values with IN operator",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.CVE),
				).
				AddRegexes(search.VulnerabilityState, ".+ED").
				ProtoQuery(),
			schema:          imageCVEsSchema,
			flattenedSchema: imageCVEV2Schema,
			expectedQuery: normalizeStatement(`select image_cves.CveBaseInfo_Cve as cve
				from image_cves
				inner join image_component_cve_edges on image_cves.Id = image_component_cve_edges.ImageCveId
				inner join image_component_edges on image_component_cve_edges.ImageComponentId = image_component_edges.ImageComponentId
				inner join images on image_component_edges.ImageId = images.Id
				inner join image_cve_edges on(image_component_edges.ImageId = image_cve_edges.ImageId and image_cves.Id = image_cve_edges.ImageCveId)
				where image_cve_edges.State IN ($1, $2)`),
			expectedFlattenedQuery: "select image_cves_v2.CveBaseInfo_Cve as cve " +
				"from image_cves_v2 " +
				"where image_cves_v2.State IN ($1, $2)",
		},
	} {
		t.Run(c.desc, func(t *testing.T) {
			ctx := c.ctx
			if c.ctx == nil {
				ctx = context.Background()
			}
			testSchema := c.schema
			if features.FlattenCVEData.Enabled() && c.flattenedSchema != nil {
				testSchema = c.flattenedSchema
			}
			actualQ, err := standardizeSelectQueryAndPopulatePath(ctx, c.q, testSchema, SELECT)
			if c.expectedError != "" {
				assert.Error(t, err, c.expectedError)
				return
			}

			assert.NoError(t, err)

			if c.q == nil {
				assert.Nil(t, actualQ)
				return
			}

			expectedQuery := c.expectedQuery
			if features.FlattenCVEData.Enabled() && c.expectedFlattenedQuery != "" {
				expectedQuery = c.expectedFlattenedQuery
			}
			actual := actualQ.AsSQL()
			assert.Equal(t, expectedQuery, actual)
		})
	}
}

func TestDeleteQueries(t *testing.T) {
	t.Parallel()

	for _, c := range []struct {
		desc          string
		ctx           context.Context
		q             *v1.Query
		expectedError string
		expectedQuery string
	}{
		{
			desc: "base schema; delete 1",
			q: search.NewQueryBuilder().
				AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			expectedQuery: "delete from deployments where deployments.Name = $1",
		},
		{
			desc: "base schema; delete",
			q: search.NewQueryBuilder().
				AddSelectFields(search.NewQuerySelect(search.DeploymentName)).ProtoQuery(),
			expectedQuery: "delete from deployments",
		},
		{
			desc: "base schema; delete w/ where",
			q: search.NewQueryBuilder().
				AddSelectFields(search.NewQuerySelect(search.DeploymentName)).
				AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			expectedQuery: "delete from deployments where deployments.Name = $1",
		},
		{
			desc: "child schema; multiple delete w/ where",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.Privileged),
					search.NewQuerySelect(search.ImageName),
				).
				AddExactMatches(search.ImageName, "stackrox").ProtoQuery(),
			expectedQuery: normalizeStatement(`delete
				from deployments inner join deployments_containers
				on deployments.Id = deployments_containers.deployments_Id
				where deployments_containers.Image_Name_FullName = $1`),
		},
		{
			desc: "base schema and child schema; delete",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.DeploymentName),
					search.NewQuerySelect(search.ImageName),
				).ProtoQuery(),
			expectedQuery: normalizeStatement(`delete
				from deployments inner join deployments_containers on deployments.Id = deployments_containers.deployments_Id`),
		},
		{
			desc: "base schema and child schema conjunction query; delete w/ where",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.DeploymentName),
					search.NewQuerySelect(search.ImageName),
				).
				AddExactMatches(search.ImageName, "stackrox").
				AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			expectedQuery: normalizeStatement(`delete
				from deployments inner join deployments_containers
				on deployments.Id = deployments_containers.deployments_Id
				where (deployments.Name = $1 and deployments_containers.Image_Name_FullName = $2)`),
		},
		{
			desc: "derived field delete",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.DeploymentName).AggrFunc(aggregatefunc.Count).Distinct(),
				).ProtoQuery(),
			expectedQuery: "delete from deployments",
		},
		{
			desc: "derived field delete w/ where",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.DeploymentName).AggrFunc(aggregatefunc.Count).Distinct(),
				).
				AddExactMatches(search.ImageName, "stackrox").
				AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			expectedQuery: normalizeStatement(`delete
				from deployments inner join deployments_containers
				on deployments.Id = deployments_containers.deployments_Id
				where (deployments.Name = $1 and deployments_containers.Image_Name_FullName = $2)`),
		},
		{
			desc:          "nil query",
			expectedQuery: "delete from deployments",
		},
		{
			desc: "base schema; delete w/ conjunction",
			q: func() *v1.Query {
				q := search.ConjunctionQuery(
					search.NewQueryBuilder().
						AddExactMatches(search.DeploymentName, "dep").ProtoQuery(),
					search.NewQueryBuilder().
						AddExactMatches(search.Namespace, "ns").ProtoQuery(),
				)
				q.Selects = []*v1.QuerySelect{search.NewQuerySelect(search.DeploymentName).Proto()}
				return q
			}(),
			expectedQuery: "delete from deployments where (deployments.Name = $1 and deployments.Namespace = $2)",
		},
		{
			desc: "base schema; delete w/ where; image scope",
			ctx: scoped.Context(context.Background(), scoped.Scope{
				ID:    "fake-image",
				Level: v1.SearchCategory_IMAGES,
			}),
			q: search.NewQueryBuilder().
				AddSelectFields(search.NewQuerySelect(search.DeploymentName)).
				AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			expectedQuery: normalizeStatement(`delete from deployments
				where deployments.Name = $1`),
		},
		{
			desc: "base schema; delete w/ multiple scopes",
			ctx: scoped.Context(context.Background(), scoped.Scope{
				ID:    uuid.NewV4().String(),
				Level: v1.SearchCategory_NAMESPACES,
				Parent: &scoped.Scope{
					ID:    uuid.NewV4().String(),
					Level: v1.SearchCategory_CLUSTERS,
				},
			}),
			q: search.NewQueryBuilder().
				AddSelectFields(search.NewQuerySelect(search.DeploymentName)).
				AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			expectedQuery: normalizeStatement(`delete from deployments
				where deployments.Name = $1`),
		},
	} {
		t.Run(c.desc, func(t *testing.T) {
			ctx := c.ctx
			if c.ctx == nil {
				ctx = context.Background()
			}

			sacCtx := sac.WithAllAccess(ctx)
			actualQ, err := standardizeQueryAndPopulatePath(sacCtx, c.q, schema.DeploymentsSchema, DELETE)
			if c.expectedError != "" {
				assert.Error(t, err, c.expectedError)
				return
			}

			assert.NoError(t, err)

			actual := actualQ.AsSQL()
			assert.Equal(t, c.expectedQuery, actual)
		})
	}
}

func TestDeleteReturningIDsQueries(t *testing.T) {
	t.Parallel()

	for _, c := range []struct {
		desc          string
		ctx           context.Context
		q             *v1.Query
		expectedError string
		expectedQuery string
	}{
		{
			desc: "base schema; delete 1",
			q: search.NewQueryBuilder().
				AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			expectedQuery: normalizeStatement(`delete from deployments where deployments.Name = $1
				returning deployments.Id::text as Deployment_ID`),
		},
		{
			desc: "base schema; delete",
			q: search.NewQueryBuilder().
				AddSelectFields(search.NewQuerySelect(search.DeploymentName)).ProtoQuery(),
			expectedQuery: normalizeStatement(`delete from deployments
				returning deployments.Id::text as Deployment_ID`),
		},
		{
			desc: "base schema; delete w/ where",
			q: search.NewQueryBuilder().
				AddSelectFields(search.NewQuerySelect(search.DeploymentName)).
				AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			expectedQuery: normalizeStatement(`delete from deployments where deployments.Name = $1
				returning deployments.Id::text as Deployment_ID`),
		},
		{
			desc: "child schema; multiple delete w/ where",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.Privileged),
					search.NewQuerySelect(search.ImageName),
				).
				AddExactMatches(search.ImageName, "stackrox").ProtoQuery(),
			expectedQuery: normalizeStatement(`delete
				from deployments inner join deployments_containers
				on deployments.Id = deployments_containers.deployments_Id
				where deployments_containers.Image_Name_FullName = $1
				returning distinct(deployments.Id::text) as Deployment_ID`),
		},
		{
			desc: "base schema and child schema; delete",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.DeploymentName),
					search.NewQuerySelect(search.ImageName),
				).ProtoQuery(),
			expectedQuery: normalizeStatement(`delete
				from deployments inner join deployments_containers on deployments.Id = deployments_containers.deployments_Id
				returning distinct(deployments.Id::text) as Deployment_ID`),
		},
		{
			desc: "base schema and child schema conjunction query; delete w/ where",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.DeploymentName),
					search.NewQuerySelect(search.ImageName),
				).
				AddExactMatches(search.ImageName, "stackrox").
				AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			expectedQuery: normalizeStatement(`delete
				from deployments inner join deployments_containers
				on deployments.Id = deployments_containers.deployments_Id
				where (deployments.Name = $1 and deployments_containers.Image_Name_FullName = $2)
				returning distinct(deployments.Id::text) as Deployment_ID`),
		},
		{
			desc: "derived field delete",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.DeploymentName).AggrFunc(aggregatefunc.Count).Distinct(),
				).ProtoQuery(),
			expectedQuery: normalizeStatement(`delete from deployments
				returning deployments.Id::text as Deployment_ID`),
		},
		{
			desc: "derived field delete w/ where",
			q: search.NewQueryBuilder().
				AddSelectFields(
					search.NewQuerySelect(search.DeploymentName).AggrFunc(aggregatefunc.Count).Distinct(),
				).
				AddExactMatches(search.ImageName, "stackrox").
				AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			expectedQuery: normalizeStatement(`delete
				from deployments inner join deployments_containers
				on deployments.Id = deployments_containers.deployments_Id
				where (deployments.Name = $1 and deployments_containers.Image_Name_FullName = $2)
				returning distinct(deployments.Id::text) as Deployment_ID`),
		},
		{
			desc: "nil query",
			expectedQuery: normalizeStatement(`delete from deployments
				returning deployments.Id::text as Deployment_ID`),
		},
		{
			desc: "base schema; delete w/ conjunction",
			q: func() *v1.Query {
				q := search.ConjunctionQuery(
					search.NewQueryBuilder().
						AddExactMatches(search.DeploymentName, "dep").ProtoQuery(),
					search.NewQueryBuilder().
						AddExactMatches(search.Namespace, "ns").ProtoQuery(),
				)
				q.Selects = []*v1.QuerySelect{search.NewQuerySelect(search.DeploymentName).Proto()}
				return q
			}(),
			expectedQuery: normalizeStatement(`delete from deployments where (deployments.Name = $1 and deployments.Namespace = $2)
				returning deployments.Id::text as Deployment_ID`),
		},
		{
			desc: "base schema; delete w/ where; image scope",
			ctx: scoped.Context(context.Background(), scoped.Scope{
				ID:    "fake-image",
				Level: v1.SearchCategory_IMAGES,
			}),
			q: search.NewQueryBuilder().
				AddSelectFields(search.NewQuerySelect(search.DeploymentName)).
				AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			expectedQuery: normalizeStatement(`delete from deployments
				where deployments.Name = $1
				returning deployments.Id::text as Deployment_ID`),
		},
		{
			desc: "base schema; delete w/ multiple scopes",
			ctx: scoped.Context(context.Background(), scoped.Scope{
				ID:    uuid.NewV4().String(),
				Level: v1.SearchCategory_NAMESPACES,
				Parent: &scoped.Scope{
					ID:    uuid.NewV4().String(),
					Level: v1.SearchCategory_CLUSTERS,
				},
			}),
			q: search.NewQueryBuilder().
				AddSelectFields(search.NewQuerySelect(search.DeploymentName)).
				AddExactMatches(search.DeploymentName, "central").ProtoQuery(),
			expectedQuery: normalizeStatement(`delete from deployments
				where deployments.Name = $1
				returning deployments.Id::text as Deployment_ID`),
		},
	} {
		t.Run(c.desc, func(t *testing.T) {
			ctx := c.ctx
			if c.ctx == nil {
				ctx = context.Background()
			}

			sacCtx := sac.WithAllAccess(ctx)
			actualQ, err := standardizeQueryAndPopulatePath(sacCtx, c.q, schema.DeploymentsSchema, DELETERETURNINGIDS)
			if c.expectedError != "" {
				assert.Error(t, err, c.expectedError)
				return
			}

			assert.NoError(t, err)

			actual := actualQ.AsSQL()
			assert.Equal(t, c.expectedQuery, actual)
		})
	}
}

func normalizeStatement(statement string) string {
	return strings.ReplaceAll(strings.ReplaceAll(statement, "\t", ""), "\n", " ")
}
