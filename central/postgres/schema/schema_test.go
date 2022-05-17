package schema

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/features"
	pkgPostgres "github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type gormTable struct {
	name     string
	instance interface{}
}

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
	if s.pool != nil {
		s.pool.Close()
	}
	s.envIsolator.RestoreAll()
}

func (s *SchemaTestSuite) TestGormConsistentWithSQL() {
	testCases := []struct {
		file        string
		gormTables  []gormTable
		createStmts *pkgPostgres.CreateStmts
	}{
		{
			file:        "alerts.go",
			createStmts: CreateTableAlertsStmt,
			gormTables: []gormTable{
				{
					name:     AlertsTableName,
					instance: Alerts{},
				},
			},
		},
	}
	/*
		{"apitokens.go"},
		{"authproviders.go"},
		{"cluster_cves.go"},
		{"cluster_health_status.go"},
		{"clusterinitbundles.go"},
		{"clusters.go"},
		{"deployments.go"},
		{"image_component_cve_relations.go"},
		{"image_component_relations.go"},
		{"image_components.go"},
		{"image_cve_relations.go"},
		{"image_cves.go"},
		{"images.go"},
		{"integrationhealth.go"},
		{"k8sroles.go"},
		{"multikey.go"},
		{"namespaces.go"},
		{"networkbaseline.go"},
		{"networkentity.go"},
		{"node_components.go"},
		{"node_components_to_cves.go"},
		{"node_cves.go"},
		{"nodes.go"},
		{"nodes_to_components.go"},
		{"notifiers.go"},
		{"permissionsets.go"},
		{"pods.go"},
		{"policy.go"},
		{"process_indicators.go"},
		{"processbaselines.go"},
		{"processwhitelistresults.go"},
		{"reportconfigs.go"},
		{"risk.go"},
		{"rolebindings.go"},
		{"roles.go"},
		{"schema_test.go"},
		{"secrets.go"},
		{"serviceaccounts.go"},
		{"signatureintegrations.go"},
		{"simpleaccessscopes.go"},
		{"singlekey.go"},
		{"testchild1.go"},
		{"testchild2.go"},
		{"testg2grandchild1.go"},
		{"testg3grandchild1.go"},
		{"testggrandchild1.go"},
		{"testgrandchild1.go"},
		{"testgrandparent.go"},
		{"testparent1.go"},
		{"testparent2.go"},
		{"testparent3.go"},
		{"vulnreq.go"},
		{"watchedimages.go"},
	} */
	for _, testCase := range testCases {
		s.T().Run(testCase.file, func(t *testing.T) {
			gormSchema := s.getGormTableSchema(testCase.gormTables[0].name, testCase.gormTables[0].instance)
			sqlSchema := s.getSQLTableSchema(testCase.gormTables[0].name, testCase.createStmts)
			s.Require().Equal(sqlSchema, gormSchema)
		})
	}
}

func (s *SchemaTestSuite) getGormTableSchema(table string, i interface{}) string {
	s.Require().NoError(s.gorm.Table(table).AutoMigrate(i))
	defer s.pool.Exec(s.ctx, fmt.Sprintf("DROP table IF EXISTS %s", table))
	return s.dumpSchema(table)
}

func (s *SchemaTestSuite) getSQLTableSchema(table string, stmt *pkgPostgres.CreateStmts) string {
	pgutils.CreateTable(s.ctx, s.pool, stmt)
	defer s.pool.Exec(s.ctx, fmt.Sprintf("DROP table IF EXISTS %s", table))
	return s.dumpSchema(table)
}

func (s *SchemaTestSuite) dumpSchema(table string) string {
	// Could use pg_commands but this will exist only for a while
	cmd := exec.Command(`pg_dump`, `--schema-only`, `--db`, `postgres`, `-t`, table)
	out, err := cmd.Output()
	s.Require().NoError(err)
	return string(out)
}
