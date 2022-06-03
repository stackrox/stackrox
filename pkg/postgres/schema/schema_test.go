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
	"gorm.io/gorm"
)

var (
	addConstraintRegex = regexp.MustCompile(`ADD CONSTRAINT (\S+) `)
	fKConstraintRegex  = regexp.MustCompile(`(\S+); Type: FK CONSTRAINT; `)
	excludeFiles       = set.NewStringSet("all.go", "schema_test.go")
)

type SchemaTestSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
	connConfig  *pgx.ConnConfig
	pool        *pgxpool.Pool
	gormDB      *gorm.DB
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
	s.gormDB = pgtest.OpenGormDB(s.T(), source)
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
	pgtest.CloseGormDB(s.T(), s.gormDB)
}

func (s *SchemaTestSuite) TestGormConsistentWithSQL() {
	type testCaseStruct struct {
		name        string
		createStmts *pkgPostgres.CreateStmts
	}
	var testCases []testCaseStruct
	for _, rt := range getAllRegisteredTablesInOrder() {
		testCases = append(testCases, testCaseStruct{rt.Schema.Table, rt.CreateStmt})
	}

	for _, testCase := range testCases {
		s.T().Run(fmt.Sprintf("check if %q schemas are equal", testCase.name), func(t *testing.T) {
			schema := GetSchemaForTable(testCase.name)
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

// TestReentry checks if we can apply the schema multiple times.
func (s *SchemaTestSuite) TestReentry() {
	ApplyAllSchemas(s.ctx, s.gormDB)
	ApplyAllSchemas(s.ctx, s.gormDB)
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
	pgutils.CreateTableFromModel(s.ctx, s.gormDB, createStmt)
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
	err := s.gormDB.Migrator().DropTable(createStmt.GormModel)
	s.Require().NoError(err)

	for _, child := range createStmt.Children {
		s.dropTableFromModel(child)
	}
	s.Require().False(s.gormDB.Migrator().HasTable(createStmt.GormModel))
}

func (s *SchemaTestSuite) tablesForSchema(schema *walker.Schema) []string {
	tables := []string{schema.Table}
	for _, child := range schema.Children {
		s.tablesForSchema(child)
	}
	return tables
}
