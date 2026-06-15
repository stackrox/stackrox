//go:build sql_integration

package schema

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/pkg/postgres"
	pkgPostgres "github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest/conn"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
	k8sEnv "k8s.io/utils/env"
)

var (
	excludeFiles = set.NewStringSet("all.go", "schema_test.go")
)

type SchemaTestSuite struct {
	suite.Suite
	connConfig *pgx.ConnConfig
	pool       postgres.DB
	gormDB     *gorm.DB
	ctx        context.Context
}

func TestSchema(t *testing.T) {
	suite.Run(t, new(SchemaTestSuite))
}

func (s *SchemaTestSuite) SetupSuite() {
	ctx := sac.WithAllAccess(context.Background())
	source := conn.GetConnectionStringWithDatabaseName(s.T(), k8sEnv.GetString("POSTGRES_DB", "postgres"))

	config, err := postgres.ParseConfig(source)
	s.NoError(err)

	s.connConfig = config.ConnConfig

	s.Require().NoError(err)
	pool, err := postgres.New(ctx, config)
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
	if s.pool == nil {
		return
	}
	s.pool.Close()
	conn.CloseGormDB(s.T(), s.gormDB)
}

func (s *SchemaTestSuite) TestTableNameSanity() {
	type testCaseStruct struct {
		name           string
		createStmts    *pkgPostgres.CreateStmts
		featureEnabled func() bool
	}
	var testCases []testCaseStruct
	for _, rt := range getAllTables() {
		testCases = append(testCases, testCaseStruct{rt.Schema.Table, rt.CreateStmt, rt.FeatureEnabledFunc})
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

func (s *SchemaTestSuite) TestApplyAllIndexes() {
	s.registerTestIndexTable()
	ApplyAllSchemas(s.ctx, s.gormDB)
	ApplyAllIndexes(s.ctx, s.pool)

	created := s.getIndexNames("idx_test_applies")
	s.Contains(created, "idxtestapply_col_a")
	s.Contains(created, "idxtestapply_composite")
	s.Contains(created, "idxtestapply_unique_col")
	s.Contains(created, "idxtestapply_bg_col")
}

func (s *SchemaTestSuite) TestApplyAllStartupIndexes() {
	s.registerTestIndexTable()
	ApplyAllSchemas(s.ctx, s.gormDB)
	err := ApplyAllStartupIndexes(s.ctx, s.pool)
	s.Require().NoError(err)

	created := s.getIndexNames("idx_test_applies")
	s.Contains(created, "idxtestapply_col_a", "non-unique startup index should be created")
	s.Contains(created, "idxtestapply_composite", "composite startup index should be created")
	s.Contains(created, "idxtestapply_unique_col", "unique startup index should be created")
	s.NotContains(created, "idxtestapply_bg_col", "background index should not be created at startup")
}

func (s *SchemaTestSuite) TestApplyAllBackgroundIndexes() {
	s.registerTestIndexTable()
	ApplyAllSchemas(s.ctx, s.gormDB)
	err := ApplyAllBackgroundIndexes(s.ctx, s.pool)
	s.Require().NoError(err)

	created := s.getIndexNames("idx_test_applies")
	s.Contains(created, "idxtestapply_bg_col", "background index should be created")
	s.NotContains(created, "idxtestapply_col_a", "startup index should not be created by background apply")
}

func (s *SchemaTestSuite) TestApplyIndexesIdempotent() {
	s.registerTestIndexTable()
	ApplyAllSchemas(s.ctx, s.gormDB)
	ApplyAllIndexes(s.ctx, s.pool)
	ApplyAllIndexes(s.ctx, s.pool)
}

type IdxTestApply struct {
	ID        string `gorm:"column:id;type:text;primaryKey"`
	ColA      string `gorm:"column:col_a;type:text"`
	ColB      string `gorm:"column:col_b;type:text"`
	UniqueCol string `gorm:"column:unique_col;type:text"`
	BgCol     string `gorm:"column:bg_col;type:text"`
}

func (s *SchemaTestSuite) registerTestIndexTable() {
	const tableName = "idx_test_applies"
	if _, exists := registeredTables[tableName]; exists {
		return
	}
	schema := &walker.Schema{
		Table: tableName,
		Type:  "storage.IdxTestApply",
	}
	stmt := &pkgPostgres.CreateStmts{
		GormModel: (*IdxTestApply)(nil),
		Indexes: []*pkgPostgres.IndexDefinition{
			{Name: "idxtestapply_col_a", CreateSQL: "CREATE INDEX CONCURRENTLY IF NOT EXISTS idxtestapply_col_a ON idx_test_applies USING btree (col_a)", Background: false},
			{Name: "idxtestapply_composite", CreateSQL: "CREATE INDEX CONCURRENTLY IF NOT EXISTS idxtestapply_composite ON idx_test_applies USING btree (col_a, col_b)", Background: false},
			{Name: "idxtestapply_unique_col", CreateSQL: "CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idxtestapply_unique_col ON idx_test_applies USING btree (unique_col)", Background: false},
			{Name: "idxtestapply_bg_col", CreateSQL: "CREATE INDEX CONCURRENTLY IF NOT EXISTS idxtestapply_bg_col ON idx_test_applies USING btree (bg_col)", Background: true},
		},
	}
	registeredTables[tableName] = &registeredTable{
		Schema:             schema,
		CreateStmt:         stmt,
		FeatureEnabledFunc: func() bool { return true },
	}
}

func (s *SchemaTestSuite) getIndexNames(table string) set.StringSet {
	rows, err := s.pool.Query(s.ctx,
		`SELECT c.relname FROM pg_index i
		 JOIN pg_class c ON c.oid = i.indexrelid
		 JOIN pg_class t ON t.oid = i.indrelid
		 WHERE t.relname = $1 AND i.indisvalid`, table)
	s.Require().NoError(err)
	defer rows.Close()

	result := set.NewStringSet()
	for rows.Next() {
		var name string
		s.Require().NoError(rows.Scan(&name))
		result.Add(name)
	}
	s.Require().NoError(rows.Err())
	return result
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
