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

var (
	addConstraintRegex = regexp.MustCompile(`ADD CONSTRAINT (\S+) `)
	fKConstraintRegex  = regexp.MustCompile(`(\S+); Type: FK CONSTRAINT; `)
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
	_, err := s.pool.Exec(s.ctx, "DROP SCHEMA public CASCADE")
	s.Require().NoError(err)
	_, err = s.pool.Exec(s.ctx, "CREATE SCHEMA public")
	s.Require().NoError(err)
	if s.pool != nil {
		s.pool.Close()
	}
	s.envIsolator.RestoreAll()
}

type Inner struct {
	Id        string
	InnerElse string
	Kle       int
}
type Product struct {
	Id          string
	ProductElse string
	InnerId     string
	Xpp         Inner `gorm:"foreignKey:InnerId;references:Id;constraint:OnDelete:CASCADE"`
}

func (s *SchemaTestSuite) TestSQL() {
	s.Require().NoError(s.gorm.AutoMigrate(&Product{}))
	s.Require().NoError(s.gorm.Table(DeploymentsTableName).AutoMigrate(&Deployments{}))
	s.Require().NoError(s.gorm.AutoMigrate(&DeploymentsContainers{}))
	fmt.Println("")
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
		{
			file:        "apitokens.go",
			createStmts: CreateTableApitokensStmt,
			gormTables: []gormTable{
				{
					name:     ApitokensTableName,
					instance: Apitokens{},
				},
			},
		},
		{
			file:        "authproviders.go",
			createStmts: CreateTableAuthprovidersStmt,
			gormTables: []gormTable{
				{
					name:     AuthprovidersTableName,
					instance: Authproviders{},
				},
			},
		},
		{
			file:        "cluster_cves.go",
			createStmts: CreateTableClusterCvesStmt,
			gormTables: []gormTable{
				{
					name:     ClusterCvesTableName,
					instance: ClusterCves{},
				},
			},
		},
		{
			file:        "cluster_health_statuses.go",
			createStmts: CreateTableClusterHealthStatusesStmt,
			gormTables: []gormTable{
				{
					name:     ClusterHealthStatusesTableName,
					instance: ClusterHealthStatuses{},
				},
			},
		},
		{
			file:        "clusterinitbundles.go",
			createStmts: CreateTableClusterinitbundlesStmt,
			gormTables: []gormTable{
				{
					name:     ClusterinitbundlesTableName,
					instance: Clusterinitbundles{},
				},
			},
		},
		{
			file:        "clusters.go",
			createStmts: CreateTableClustersStmt,
			gormTables: []gormTable{
				{
					name:     ClustersTableName,
					instance: Clusters{},
				},
			},
		},
		{
			file:        "deployments.go",
			createStmts: CreateTableDeploymentsStmt,
			gormTables: []gormTable{
				{
					name:     DeploymentsTableName,
					instance: Deployments{},
				},
				{
					name:     DeploymentsContainersTableName,
					instance: DeploymentsContainers{},
				},
				{
					name:     DeploymentsContainersEnvTableName,
					instance: DeploymentsContainersEnv{},
				}, /*
					{
						name:     DeploymentsContainersVolumesTableName,
						instance: DeploymentsContainersVolumes{},
					},
					{
						name:     DeploymentsContainersSecretsTableName,
						instance: DeploymentsContainersSecrets{},
					},
					{
						name:     DeploymentsPortsTableName,
						instance: DeploymentsPorts{},
					},
					{
						name:     DeploymentsPortsExposureInfosTableName,
						instance: DeploymentsPortsExposureInfos{},
					},*/
			},
		},
		{
			file:        "images",
			createStmts: CreateTableImagesStmt,
			gormTables: []gormTable{
				{
					name:     ImagesTableName,
					instance: Images{},
				},
				{
					name:     ImagesLayersTableName,
					instance: ImagesLayers{},
				},
			},
		}, /*
			{
				file:        "image_component_cve_relations.go",
				createStmts: CreateTableImageComponentCveRelationsStmt,
				gormTables: []gormTable{
					{
						name:     ImageComponentRelationsTableName,
						instance: ImageComponentRelations{},
					},
				},
			},
				{
					file: "",
					{"image_component_relations.go"},
					createStmts: CreateTable,
					gormTables: []gormTable{
						{
							name:     TableName,
							instance: {},
						},
					},
				},
				{
					file: {"image_components.go"},
					"",
					createStmts: CreateTable,
					gormTables: []gormTable{
						{
							name:     TableName,
							instance: {},
						},
					},
				},
				{
					file: "",
					{"image_cve_relations.go"},
					createStmts: CreateTable,
					gormTables: []gormTable{
						{
							name:     TableName,
							instance: {},
						},
					},
				},
				{
					file:        "",
					createStmts: CreateTable,
					gormTables: []gormTable{
						{
							name:     TableName,
							instance: {},
						},
					},
				},
		*/
	}
	/*
		{"image_cves.go"},
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
			gormSchemas := s.getGormTableSchemas(testCase.gormTables)
			pgutils.CreateTable(s.ctx, s.pool, testCase.createStmts)
			for table, gormSchema := range gormSchemas {
				sqlSchema := s.dumpSchema(table)
				s.Require().Equal(sqlSchema, gormSchema)
			}
			// s.Require().Len(testCase.gormTables, len(testCase.createStmts.Children)+1)
		})
	}
}

func (s *SchemaTestSuite) getGormTableSchemas(gormTables []gormTable) map[string]string {
	var tables []string
	for _, tbl := range gormTables {
		tables = append(tables, tbl.name)
		s.Require().NoError(s.gorm.AutoMigrate(tbl.instance))
	}
	defer s.pool.Exec(s.ctx, fmt.Sprintf("DROP table IF EXISTS %s", strings.Join(tables, ",")))

	tableMap := make(map[string]string, len(gormTables))
	for _, tbl := range gormTables {
		tableMap[tbl.name] = s.dumpSchema(tbl.name)
	}
	return tableMap
}

func (s *SchemaTestSuite) dumpSchema(table string) string {
	// Could use pg_commands but this will exist only for a while
	cmd := exec.Command(`pg_dump`, `--schema-only`, `--db`, `postgres`, `-t`, table)
	out, err := cmd.Output()
	s.Require().NoError(err)
	return fKConstraintRegex.ReplaceAllString(addConstraintRegex.ReplaceAllString(string(out), ""), "")
}
