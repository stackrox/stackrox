package schema

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/features"
	pkgPostgres "github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	addConstraintRegex = regexp.MustCompile(`ADD CONSTRAINT (\S+) `)
	fKConstraintRegex  = regexp.MustCompile(`(\S+); Type: FK CONSTRAINT; `)
	excludeFiles       = set.NewStringSet("schema_test.go")
)

type SchemaTestSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
	pool        *pgxpool.Pool
	gorm        *gorm.DB
	ctx         context.Context
	tmpDir      string
}

func TestSchema(t *testing.T) {
	suite.Run(t, new(SchemaTestSuite))
}

func (s *SchemaTestSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")

	if !features.PostgresDatastore.Enabled() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}

	ctx := sac.WithAllAccess(context.Background())

	source := pgtest.GetConnectionString(s.T())
	config, err := pgxpool.ParseConfig(source)

	s.Require().NoError(err)
	pool, err := pgxpool.ConnectConfig(ctx, config)
	s.Require().NoError(err)

	s.ctx = ctx
	s.pool = pool
	s.tmpDir, err = os.MkdirTemp("", "schema_test")
	s.Require().NoError(err)
	source = "host=localhost port=5432 database=postgres user=cong password= sslmode=disable statement_timeout=600000"
	s.gorm, err = gorm.Open(postgres.Open(source), &gorm.Config{})
	s.Require().NoError(err)
}

func (s *SchemaTestSuite) TearDownTest() {
	_, err := s.pool.Exec(s.ctx, "DROP SCHEMA public CASCADE")
	s.Require().NoError(err)
	_, err = s.pool.Exec(s.ctx, "CREATE SCHEMA public")
	s.Require().NoError(err)
	if s.pool != nil {
		s.pool.Close()
	}
	s.envIsolator.RestoreAll()
}

func (s *SchemaTestSuite) TestGormConsistentWithSQL() {
	allTestCases := set.NewStringSet(s.getAllTestCases()...)
	testCases := []struct {
		name        string
		createStmts *pkgPostgres.CreateStmts
	}{
		{
			name:        AlertsTableName,
			createStmts: CreateTableAlertsStmt,
		},
		{
			name:        ApiTokensTableName,
			createStmts: CreateTableApiTokensStmt,
		},
		{
			name:        AuthProvidersTableName,
			createStmts: CreateTableAuthProvidersStmt,
		},
		{
			name:        ClusterCvesTableName,
			createStmts: CreateTableClusterCvesStmt,
		},
		{
			name:        ClusterHealthStatusesTableName,
			createStmts: CreateTableClusterHealthStatusesStmt,
		},
		{
			name:        ClusterInitBundlesTableName,
			createStmts: CreateTableClusterInitBundlesStmt,
		},
		{
			name:        ClustersTableName,
			createStmts: CreateTableClustersStmt,
		},
		{
			name:        DeploymentsTableName,
			createStmts: CreateTableDeploymentsStmt,
		},
		{
			name:        ImagesTableName,
			createStmts: CreateTableImagesStmt,
		},
		{
			name:        ImageComponentsTableName,
			createStmts: CreateTableImageComponentsStmt,
		},
		{
			name:        ImageComponentCveEdgesTableName,
			createStmts: CreateTableImageComponentCveEdgesStmt,
		},
		{
			name:        ImageComponentEdgesTableName,
			createStmts: CreateTableImageComponentEdgesStmt,
		},
		{
			name:        ImageCveEdgesTableName,
			createStmts: CreateTableImageCveEdgesStmt,
		},
		{
			name:        ImageCvesTableName,
			createStmts: CreateTableImageCvesStmt,
		},
		{
			name:        IntegrationHealthsTableName,
			createStmts: CreateTableIntegrationHealthsStmt,
		},
		{
			name:        K8sRolesTableName,
			createStmts: CreateTableK8sRolesStmt,
		},
		{
			name:        TestMultiKeyStructsTableName,
			createStmts: CreateTableTestMultiKeyStructsStmt,
		},
		{
			name:        NamespacesTableName,
			createStmts: CreateTableNamespacesStmt,
		},
		{
			name:        NetworkBaselinesTableName,
			createStmts: CreateTableNetworkBaselinesStmt,
		},
		{
			name:        NetworkEntitiesTableName,
			createStmts: CreateTableNetworkEntitiesStmt,
		},
		{
			name:        NodeComponentsTableName,
			createStmts: CreateTableNodeComponentsStmt,
		},
		{
			name:        NodeComponentCveEdgesTableName,
			createStmts: CreateTableNodeComponentCveEdgesStmt,
		},
		{
			name:        NodeCvesTableName,
			createStmts: CreateTableNodeCvesStmt,
		},
		{
			name:        NodesTableName,
			createStmts: CreateTableNodesStmt,
		},
		{
			name:        NodeComponentEdgesTableName,
			createStmts: CreateTableNodeComponentEdgesStmt,
		},
		{
			name:        NotifiersTableName,
			createStmts: CreateTableNotifiersStmt,
		},
		{
			name:        PermissionSetsTableName,
			createStmts: CreateTablePermissionSetsStmt,
		},
		{
			name:        PodsTableName,
			createStmts: CreateTablePodsStmt,
		},
		{
			name:        PoliciesTableName,
			createStmts: CreateTablePoliciesStmt,
		},
		{
			name:        ProcessBaselineResultsTableName,
			createStmts: CreateTableProcessBaselineResultsStmt,
		},
		{
			name:        ProcessBaselinesTableName,
			createStmts: CreateTableProcessBaselinesStmt,
		},
		{
			name:        ProcessIndicatorsTableName,
			createStmts: CreateTableProcessIndicatorsStmt,
		},
		{
			name:        ReportConfigurationsTableName,
			createStmts: CreateTableReportConfigurationsStmt,
		},
		{
			name:        RisksTableName,
			createStmts: CreateTableRisksStmt,
		},
		{
			name:        RoleBindingsTableName,
			createStmts: CreateTableRoleBindingsStmt,
		},
		{
			name:        RolesTableName,
			createStmts: CreateTableRolesStmt,
		},
		{
			name:        SecretsTableName,
			createStmts: CreateTableSecretsStmt,
		},
		{
			name:        ServiceAccountsTableName,
			createStmts: CreateTableServiceAccountsStmt,
		},
		{
			name:        SignatureIntegrationsTableName,
			createStmts: CreateTableSignatureIntegrationsStmt,
		},
		{
			name:        SimpleAccessScopesTableName,
			createStmts: CreateTableSimpleAccessScopesStmt,
		},
		{
			name:        VulnerabilityRequestsTableName,
			createStmts: CreateTableVulnerabilityRequestsStmt,
		},
		{
			name:        WatchedImagesTableName,
			createStmts: CreateTableWatchedImagesStmt,
		},
		{
			name:        TestSingleKeyStructsTableName,
			createStmts: CreateTableTestSingleKeyStructsStmt,
		},

		{
			name:        TestChild1TableName,
			createStmts: CreateTableTestChild1Stmt,
		},
		{
			name:        TestGrandparentsTableName,
			createStmts: CreateTableTestGrandparentsStmt,
		},
		{
			name:        TestParent1TableName,
			createStmts: CreateTableTestParent1Stmt,
		},

		{
			name:        TestParent3TableName,
			createStmts: CreateTableTestParent3Stmt,
		},
		{
			name:        TestParent2TableName,
			createStmts: CreateTableTestParent2Stmt,
		},
		{
			name:        TestGrandChild1TableName,
			createStmts: CreateTableTestGrandChild1Stmt,
		},
		{
			name:        TestG2GrandChild1TableName,
			createStmts: CreateTableTestG2GrandChild1Stmt,
		},
		{
			name:        TestG3GrandChild1TableName,
			createStmts: CreateTableTestG3GrandChild1Stmt,
		},
		{
			name:        TestGGrandChild1TableName,
			createStmts: CreateTableTestGGrandChild1Stmt,
		},
		{
			name:        TestChild2TableName,
			createStmts: CreateTableTestChild2Stmt,
		},
	}
	for _, testCase := range testCases {
		s.T().Run(testCase.name, func(t *testing.T) {
			s.Require().Contains(allTestCases, testCase.name)
			allTestCases.Remove(testCase.name)
			schema := globaldb.GetSchemaForTable(testCase.name)
			gormSchemas := s.getGormTableSchemas(schema, testCase.createStmts)
			pgutils.CreateTable(s.ctx, s.pool, testCase.createStmts)
			for table, gormSchema := range gormSchemas {
				sqlSchema := s.dumpSchema(table)
				s.Require().Equal(sqlSchema, gormSchema)
			}
			// Check if the table name is reversible.
			// Gorm may have wrong behavior if the table name is not reversible.
			schemaName := pgutils.NamingStrategy.SchemaName(testCase.name)
			s.Require().Equal(testCase.name, pgutils.NamingStrategy.TableName(schemaName))
		})
	}
	s.Require().Len(allTestCases, 0)
}

func (s *SchemaTestSuite) getAllTestCases() []string {
	files, err := os.ReadDir(".")
	s.Require().NoError(err)
	var testCases []string
	for _, file := range files {
		name := file.Name()
		if excludeFiles.Contains(name) || !strings.HasSuffix(name, ".go") {
			fmt.Printf("Skipping %s\n", name)
			continue
		}
		testCases = append(testCases, strings.TrimSuffix(name, ".go"))
	}
	return testCases
}

func (s *SchemaTestSuite) getGormTableSchemas(schema *walker.Schema, createStmt *pkgPostgres.CreateStmts) map[string]string {
	pgutils.CreateTableFromModel(s.gorm, createStmt)
	defer s.dropTableFromModel(createStmt)
	tables := s.tablesForSchema(schema)

	tableMap := make(map[string]string, len(tables))
	for _, tbl := range tables {
		tableMap[tbl] = s.dumpSchema(tbl)
	}
	return tableMap
}

func (s *SchemaTestSuite) dumpSchema(table string) string {
	// Dump Postgres schema
	cmd := exec.Command(`pg_dump`, `--schema-only`, `--db`, `postgres`, `-t`, table)
	out, err := cmd.Output()
	s.Require().NoError(err)
	return fKConstraintRegex.ReplaceAllString(addConstraintRegex.ReplaceAllString(string(out), ""), "")
}

func (s *SchemaTestSuite) dropTableFromModel(createStmt *pkgPostgres.CreateStmts) {
	err := s.gorm.Migrator().DropTable(createStmt.GormModel)
	s.Require().NoError(err)

	for _, child := range createStmt.Children {
		s.dropTableFromModel(child)
	}
	s.Require().False(s.gorm.Migrator().HasTable(createStmt.GormModel))
}

func (s *SchemaTestSuite) tablesForSchema(schema *walker.Schema) []string {
	tables := []string{schema.Table}
	for _, child := range schema.Children {
		s.tablesForSchema(child)
	}
	return tables
}
