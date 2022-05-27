//go:build sql_integration
// +build sql_integration

package schema

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/jackc/pgx/v4"
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
	"github.com/stretchr/testify/assert"
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
	connConfig  *pgx.ConnConfig
	pool        *pgxpool.Pool
	gorm        *gorm.DB
	ctx         context.Context
}

func TestSchema(t *testing.T) {
	suite.Run(t, new(SchemaTestSuite))
}

func (s *SchemaTestSuite) SetupSuite() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")

	if !features.PostgresDatastore.Enabled() {
		s.T().Skip("Skip postgres store tests")
	}

	ctx := sac.WithAllAccess(context.Background())

	source := pgtest.GetConnectionString(s.T())
	config, err := pgxpool.ParseConfig(source)
	s.connConfig = config.ConnConfig

	s.Require().NoError(err)
	pool, err := pgxpool.ConnectConfig(ctx, config)
	s.Require().NoError(err)

	s.ctx = ctx
	s.pool = pool
	s.Require().NoError(err)
	s.gorm, err = gorm.Open(postgres.Open(source), &gorm.Config{})
	s.Require().NoError(err)
}

func (s *SchemaTestSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
	if s.pool == nil {
		return
	}
	_, err := s.pool.Exec(s.ctx, "DROP SCHEMA public CASCADE")
	s.Require().NoError(err)
	_, err = s.pool.Exec(s.ctx, "CREATE SCHEMA public")
	s.Require().NoError(err)
}

func (s *SchemaTestSuite) TearDownSuite() {
	s.envIsolator.RestoreAll()
	if s.pool == nil {
		return
	}
	s.pool.Close()
}

func (s *SchemaTestSuite) TestGormConsistentWithSQL() {
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
			name:        ClustersTableName,
			createStmts: CreateTableClustersStmt,
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
			name:        ComplianceoperatorcheckresultsTableName,
			createStmts: CreateTableComplianceoperatorcheckresultsStmt,
		},
		{
			name:        ComplianceoperatorprofilesTableName,
			createStmts: CreateTableComplianceoperatorprofilesStmt,
		},
		{
			name:        ComplianceoperatorrulesTableName,
			createStmts: CreateTableComplianceoperatorrulesStmt,
		},
		{
			name:        ComplianceoperatorscansTableName,
			createStmts: CreateTableComplianceoperatorscansStmt,
		},
		{
			name:        ComplianceoperatorscansettingbindingsTableName,
			createStmts: CreateTableComplianceoperatorscansettingbindingsStmt,
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
			name:        NetworkGraphConfigsTableName,
			createStmts: CreateTableNetworkGraphConfigsStmt,
		},
		{
			name:        NetworkpoliciesTableName,
			createStmts: CreateTableNetworkpoliciesStmt,
		},
		{
			name:        NetworkpolicyapplicationundorecordsTableName,
			createStmts: CreateTableNetworkpolicyapplicationundorecordsStmt,
		},
		{
			name:        NetworkpoliciesundodeploymentsTableName,
			createStmts: CreateTableNetworkpoliciesundodeploymentsStmt,
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
		s.T().Run(fmt.Sprintf("check if %q schemas are equal", testCase.name), func(t *testing.T) {
			schema := globaldb.GetSchemaForTable(testCase.name)
			gormSchemas := s.getGormTableSchemas(schema, testCase.createStmts)
			pgutils.CreateTable(s.ctx, s.pool, testCase.createStmts)
			for table, gormSchema := range gormSchemas {
				sqlSchema := s.dumpSchema(table)
				assert.Equal(t, sqlSchema, gormSchema)
			}
		})
		s.T().Run(fmt.Sprintf("check if %q name is reversible", testCase.name), func(t *testing.T) {
			// Gorm may have wrong behavior if the table name is not reversible.
			schemaName := pgutils.NamingStrategy.SchemaName(testCase.name)
			assert.Equal(t, testCase.name, pgutils.NamingStrategy.TableName(schemaName))
		})
	}
	s.T().Run("should cover all test cases", func(t *testing.T) {
		allTestCases := set.NewStringSet(s.getAllTestCases()...)
		testCasesNames := set.NewStringSet()
		for _, testCase := range testCases {
			testCasesNames.Add(testCase.name)
		}
		assert.Equal(t, allTestCases, testCasesNames)
	})
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
	cmd := exec.Command(`pg_dump`, `--schema-only`,
		"-d", s.connConfig.Database,
		"-h", s.connConfig.Host,
		"-U", s.connConfig.User,
		"-p", fmt.Sprintf("%d", s.connConfig.Port),
		"-t", table,
		"--no-password", // never prompt for password
	)
	cmd.Env = append(cmd.Env, fmt.Sprintf("PGPASSWORD=%s", s.connConfig.Password))
	out, err := cmd.Output()
	s.Require().NoError(err, fmt.Sprintf("Failed to get schema dump\n output: %s\n err: %v\n", out, err))
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
