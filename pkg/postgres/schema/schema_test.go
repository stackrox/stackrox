//go:build sql_integration
// +build sql_integration

package schema

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/env"
	pkgPostgres "github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest/conn"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

var (
	excludeFiles = set.NewStringSet("all.go", "schema_test.go")
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
	s.envIsolator.Setenv(env.PostgresDatastoreEnabled.EnvVar(), "true")

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		s.T().Skip("Skip postgres store tests")
	}

	ctx := sac.WithAllAccess(context.Background())

	source := conn.GetConnectionString(s.T())

	config, err := pgxpool.ParseConfig(source)
	s.NoError(err)

	s.connConfig = config.ConnConfig

	s.Require().NoError(err)
	pool, err := pgxpool.ConnectConfig(ctx, config)
	s.Require().NoError(err)

	s.ctx = ctx
	s.pool = pool
	s.Require().NoError(err)
	s.gormDB = conn.OpenGormDB(s.T(), source, false)

	_, err = s.pool.Exec(s.ctx, "DROP SCHEMA public CASCADE")
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
	conn.CloseGormDB(s.T(), s.gormDB)
}

func (s *SchemaTestSuite) TestTableNameSanity() {
	type testCaseStruct struct {
		name        string
		createStmts *pkgPostgres.CreateStmts
	}
	var testCases []testCaseStruct
	for _, rt := range getAllRegisteredTablesInOrder() {
		testCases = append(testCases, testCaseStruct{rt.Schema.Table, rt.CreateStmt})
	}

	for _, testCase := range testCases {
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
