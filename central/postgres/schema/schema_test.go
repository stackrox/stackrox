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
	"gorm.io/gorm/schema"
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
	ns := schema.NamingStrategy{}
	fmt.Println(ns.TableName("TestGGrandChild1"))
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
			createStmts: CreateTableApiTokensStmt,
			gormTables: []gormTable{
				{
					name:     ApiTokensTableName,
					instance: ApiTokens{},
				},
			},
		},
		{
			file:        "authproviders.go",
			createStmts: CreateTableAuthProvidersStmt,
			gormTables: []gormTable{
				{
					name:     AuthProvidersTableName,
					instance: AuthProviders{},
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
			file:        "cluster_init_bundles.go",
			createStmts: CreateTableClusterInitBundlesStmt,
			gormTables: []gormTable{
				{
					name:     ClusterInitBundlesTableName,
					instance: ClusterInitBundles{},
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
					name:     DeploymentsContainersEnvsTableName,
					instance: DeploymentsContainersEnvs{},
				},
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
				},
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
		},
		{
			file:        "image_components.go",
			createStmts: CreateTableImageComponentsStmt,
			gormTables: []gormTable{
				{
					name:     ImageComponentsTableName,
					instance: ImageComponents{},
				},
			},
		},
		{
			file:        "image_component_cve_relations.go",
			createStmts: CreateTableImageComponentCveEdgesStmt,
			gormTables: []gormTable{
				{
					name:     ImageComponentCveEdgesTableName,
					instance: ImageComponentCveEdges{},
				},
			},
		},
		{
			file:        "image_component_relations.go",
			createStmts: CreateTableImageComponentEdgesStmt,
			gormTables: []gormTable{
				{
					name:     ImageComponentEdgesTableName,
					instance: &ImageComponentEdges{},
				},
			},
		},
		{
			file:        "image_cve_relations.go",
			createStmts: CreateTableImageCveEdgesStmt,
			gormTables: []gormTable{
				{
					name:     ImageCveEdgesTableName,
					instance: &ImageCveEdges{},
				},
			},
		},
		{
			file:        "image_cves.go",
			createStmts: CreateTableImageCvesStmt,
			gormTables: []gormTable{
				{
					name:     ImageCvesTableName,
					instance: ImageCves{},
				},
			},
		},
		{
			file:        "integration_health.go",
			createStmts: CreateTableIntegrationHealthsStmt,
			gormTables: []gormTable{
				{
					name:     IntegrationHealthsTableName,
					instance: &IntegrationHealths{},
				},
			},
		},
		{
			file:        "k8s_roles.go",
			createStmts: CreateTableK8sRolesStmt,
			gormTables: []gormTable{
				{
					name:     K8sRolesTableName,
					instance: K8sRoles{},
				},
			},
		},
		{
			file:        "multi_keys.go",
			createStmts: CreateTableTestMultiKeyStructsStmt,
			gormTables: []gormTable{
				{
					name:     TestMultiKeyStructsTableName,
					instance: TestMultiKeyStructs{},
				},
				{
					name:     TestMultiKeyStructsNestedsTableName,
					instance: TestMultiKeyStructsNesteds{},
				},
			},
		},
		{
			file:        "namespaces.go",
			createStmts: CreateTableNamespaceMetadataStmt,
			gormTables: []gormTable{
				{
					name:     NamespaceMetadataTableName,
					instance: NamespaceMetadata{},
				},
			},
		},
		{
			file:        "network_baselines.go",
			createStmts: CreateTableNetworkBaselinesStmt,
			gormTables: []gormTable{
				{
					name:     NetworkBaselinesTableName,
					instance: NetworkBaselines{},
				},
			},
		},
		{
			file:        "network_entities.go",
			createStmts: CreateTableNetworkEntitiesStmt,
			gormTables: []gormTable{
				{
					name:     NetworkEntitiesTableName,
					instance: NetworkEntities{},
				},
			},
		},
		{
			file:        "node_components.go",
			createStmts: CreateTableNodeComponentsStmt,
			gormTables: []gormTable{
				{
					name:     NodeComponentsTableName,
					instance: NodeComponents{},
				},
			},
		},
		{
			file:        "node_components_to_cves.go",
			createStmts: CreateTableNodeComponentCveEdgesStmt,
			gormTables: []gormTable{
				{
					name:     NodeComponentCveEdgesTableName,
					instance: NodeComponentCveEdges{},
				},
			},
		},
		{
			file:        "node_cves.go",
			createStmts: CreateTableNodeCvesStmt,
			gormTables: []gormTable{
				{
					name:     NodeCvesTableName,
					instance: NodeCves{},
				},
			},
		},
		{
			file:        "nodes.go",
			createStmts: CreateTableNodesStmt,
			gormTables: []gormTable{
				{
					name:     NodesTableName,
					instance: Nodes{},
				},
				{
					name:     NodesTaintsTableName,
					instance: NodesTaints{},
				},
				{
					name:     NodesComponentsTableName,
					instance: NodesComponents{},
				},
				{
					name:     NodesComponentsVulnsTableName,
					instance: NodesComponentsVulns{},
				},
			},
		},
		{
			file:        "nodes_to_components.go",
			createStmts: CreateTableNodeComponentEdgesStmt,
			gormTables: []gormTable{
				{
					name:     NodeComponentEdgesTableName,
					instance: NodeComponentEdges{},
				},
			},
		},
		{
			file:        "notifiers.go",
			createStmts: CreateTableNotifiersStmt,
			gormTables: []gormTable{
				{
					name:     NotifiersTableName,
					instance: Notifiers{},
				},
			},
		},
		{
			file:        "permission_sets.go",
			createStmts: CreateTablePermissionSetsStmt,
			gormTables: []gormTable{
				{
					name:     PermissionSetsTableName,
					instance: PermissionSets{},
				},
			},
		},
		{
			file:        "pods.go",
			createStmts: CreateTablePodsStmt,
			gormTables: []gormTable{
				{
					name:     PodsTableName,
					instance: Pods{},
				},
				{
					name:     PodsLiveInstancesTableName,
					instance: PodsLiveInstances{},
				},
			},
		},
		{
			file:        "policies.go",
			createStmts: CreateTablePoliciesStmt,
			gormTables: []gormTable{
				{
					name:     PoliciesTableName,
					instance: Policies{},
				},
			},
		},
		{
			file:        "process_indicators.go",
			createStmts: CreateTableProcessIndicatorsStmt,
			gormTables: []gormTable{
				{
					name:     ProcessIndicatorsTableName,
					instance: ProcessIndicators{},
				},
			},
		},
		{
			file:        "process_baselines.go",
			createStmts: CreateTableProcessBaselinesStmt,
			gormTables: []gormTable{
				{
					name:     ProcessBaselinesTableName,
					instance: ProcessBaselines{},
				},
			},
		},
		{
			file:        "process_whitelist_results.go",
			createStmts: CreateTableProcessWhitelistResultsStmt,
			gormTables: []gormTable{
				{
					name:     ProcessWhitelistResultsTableName,
					instance: ProcessWhitelistResults{},
				},
			},
		},
		{
			file:        "report_configs.go",
			createStmts: CreateTableReportConfigsStmt,
			gormTables: []gormTable{
				{
					name:     ReportConfigsTableName,
					instance: ReportConfigs{},
				},
			},
		},
		{
			file:        "risks.go",
			createStmts: CreateTableRisksStmt,
			gormTables: []gormTable{
				{
					name:     RisksTableName,
					instance: Risks{},
				},
			},
		},
		{
			file:        "k8s_role_bindings.go",
			createStmts: CreateTableK8sRoleBindingsStmt,
			gormTables: []gormTable{
				{
					name:     K8sRoleBindingsTableName,
					instance: K8sRoleBindings{},
				},
				{
					name:     K8sRoleBindingsSubjectsTableName,
					instance: K8sRoleBindingsSubjects{},
				},
			},
		},
		{
			file:        "roles.go",
			createStmts: CreateTableRolesStmt,
			gormTables: []gormTable{
				{
					name:     RolesTableName,
					instance: Roles{},
				},
			},
		},
		{
			file:        "secrets.go",
			createStmts: CreateTableSecretsStmt,
			gormTables: []gormTable{
				{
					name:     SecretsTableName,
					instance: Secrets{},
				},
				{
					name:     SecretsFilesTableName,
					instance: SecretsFiles{},
				},
				{
					name:     SecretsFilesRegistriesTableName,
					instance: SecretsFilesRegistries{},
				},
			},
		},
		{
			file:        "service_accounts.go",
			createStmts: CreateTableServiceAccountsStmt,
			gormTables: []gormTable{
				{
					name:     ServiceAccountsTableName,
					instance: ServiceAccounts{},
				},
			},
		},
		{
			file:        "signature_integrations.go",
			createStmts: CreateTableSignatureIntegrationsStmt,
			gormTables: []gormTable{
				{
					name:     SignatureIntegrationsTableName,
					instance: SignatureIntegrations{},
				},
			},
		},
		{
			file:        "simple_access_scopes.go",
			createStmts: CreateTableSimpleAccessScopesStmt,
			gormTables: []gormTable{
				{
					name:     SimpleAccessScopesTableName,
					instance: SimpleAccessScopes{},
				},
			},
		},
		{
			file:        "vulnerability_requests.go",
			createStmts: CreateTableVulnerabilityRequestsStmt,
			gormTables: []gormTable{
				{
					name:     VulnerabilityRequestsTableName,
					instance: VulnerabilityRequests{},
				},
				{
					name:     VulnerabilityRequestsApproversTableName,
					instance: VulnerabilityRequestsApprovers{},
				},
				{
					name:     VulnerabilityRequestsCommentsTableName,
					instance: VulnerabilityRequestsComments{},
				},
			},
		},
		{
			file:        "watched_images.go",
			createStmts: CreateTableWatchedImagesStmt,
			gormTables: []gormTable{
				{
					name:     WatchedImagesTableName,
					instance: WatchedImages{},
				},
			},
		},
		{
			file:        "test_single_key_structs.go",
			createStmts: CreateTableTestSingleKeyStructsStmt,
			gormTables: []gormTable{
				{
					name:     TestSingleKeyStructsTableName,
					instance: TestSingleKeyStructs{},
				},
			},
		},

		{
			file:        "test_child1.go",
			createStmts: CreateTableTestChild1Stmt,
			gormTables: []gormTable{
				{
					name:     TestChild1TableName,
					instance: TestChild1{},
				},
			},
		},
		{
			file:        "test_grandparents.go",
			createStmts: CreateTableTestGrandparentsStmt,
			gormTables: []gormTable{
				{
					name:     TestGrandparentsTableName,
					instance: TestGrandparents{},
				},
				{
					name:     TestGrandparentsEmbeddedsTableName,
					instance: TestGrandparentsEmbeddeds{},
				},
				{
					name:     TestGrandparentsEmbeddedsEmbedded2TableName,
					instance: TestGrandparentsEmbeddedsEmbedded2{},
				},
			},
		},
		{
			file:        "test_parent1.go",
			createStmts: CreateTableTestParent1Stmt,
			gormTables: []gormTable{
				{
					name:     TestParent1TableName,
					instance: TestParent1{},
				},
				{
					name:     TestParent1ChildrensTableName,
					instance: TestParent1Childrens{},
				},
			},
		},

		{
			file:        "test_parent3.go",
			createStmts: CreateTableTestParent3Stmt,
			gormTables: []gormTable{
				{
					name:     TestParent3TableName,
					instance: TestParent3{},
				},
			},
		},
		{
			file:        "test_parent2.go",
			createStmts: CreateTableTestParent2Stmt,
			gormTables: []gormTable{
				{
					name:     TestParent2TableName,
					instance: TestParent2{},
				},
			},
		},
		{
			file:        "test_grand_child1.go",
			createStmts: CreateTableTestGrandChild1Stmt,
			gormTables: []gormTable{
				{
					name:     TestGrandChild1TableName,
					instance: TestGrandChild1{},
				},
			},
		},
		{
			file:        "test_g2_grand_child1.go",
			createStmts: CreateTableTestG2GrandChild1Stmt,
			gormTables: []gormTable{
				{
					name:     TestG2GrandChild1TableName,
					instance: TestG2GrandChild1{},
				},
			},
		},
		{
			file:        "test_g3_grand_child1.go",
			createStmts: CreateTableTestG3GrandChild1Stmt,
			gormTables: []gormTable{
				{
					name:     TestG3GrandChild1TableName,
					instance: TestG3GrandChild1{},
				},
			},
		},
		{
			file:        "test_g_grand_child1.go",
			createStmts: CreateTableTestGGrandChild1Stmt,
			gormTables: []gormTable{
				{
					name:     TestGGrandChild1TableName,
					instance: TestGGrandChild1{},
				},
			},
		},
		{
			file:        "test_child2.go",
			createStmts: CreateTableTestChild2Stmt,
			gormTables: []gormTable{
				{
					name:     TestChild2TableName,
					instance: TestChild2{},
				},
			},
		},
	}
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
