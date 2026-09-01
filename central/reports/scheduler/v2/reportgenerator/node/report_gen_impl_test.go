//go:build sql_integration

package node

import (
	"context"
	"fmt"
	"testing"
	"time"

	clusterDSMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	nodeCVEDS "github.com/stackrox/rox/central/cve/node/datastore"
	namespaceDSMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	nodeDS "github.com/stackrox/rox/central/node/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	postgresSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type nodeReportData struct {
	clusterNames   []string
	nodeNames      []string
	componentNames []string
	cveNames       []string
}

func TestNodeReportGenerator(t *testing.T) {
	suite.Run(t, new(NodeReportGeneratorTestSuite))
}

type NodeReportGeneratorTestSuite struct {
	suite.Suite

	ctx                context.Context
	testDB             *pgtest.TestPostgres
	clusterDatastore   *clusterDSMocks.MockDataStore
	namespaceDatastore *namespaceDSMocks.MockDataStore
	nodeDatastore      nodeDS.DataStore
	nodeCVEDatastore   nodeCVEDS.DataStore
	reportGenerator    *nodeReportGeneratorImpl

	clusters   []*storage.Cluster
	namespaces []*storage.NamespaceMetadata
}

func (s *NodeReportGeneratorTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	mockCtrl := gomock.NewController(s.T())
	s.testDB = pgtest.ForT(s.T())

	s.clusterDatastore = clusterDSMocks.NewMockDataStore(mockCtrl)
	s.namespaceDatastore = namespaceDSMocks.NewMockDataStore(mockCtrl)

	s.nodeDatastore = nodeDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	var err error
	s.nodeCVEDatastore, err = nodeCVEDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.Require().NoError(err)

	s.clusters = []*storage.Cluster{
		{Id: uuid.NewV4().String(), Name: "c1"},
		{Id: uuid.NewV4().String(), Name: "c2"},
	}
	s.namespaces = []*storage.NamespaceMetadata{
		{Id: uuid.NewV4().String(), Name: "ns1", ClusterId: s.clusters[0].GetId(), ClusterName: "c1"},
		{Id: uuid.NewV4().String(), Name: "ns2", ClusterId: s.clusters[1].GetId(), ClusterName: "c2"},
	}
	s.clusterDatastore.EXPECT().GetClusters(gomock.Any()).
		Return(s.clusters, nil).AnyTimes()
	s.namespaceDatastore.EXPECT().GetAllNamespaces(gomock.Any()).
		Return(s.namespaces, nil).AnyTimes()

	nodes := s.testNodes()
	for _, node := range nodes {
		s.Require().NoError(s.nodeDatastore.UpsertNode(s.ctx, node))
	}

	s.reportGenerator = &nodeReportGeneratorImpl{
		clusterDatastore:   s.clusterDatastore,
		namespaceDatastore: s.namespaceDatastore,
		nodeCVEDatastore:   s.nodeCVEDatastore,
		db:                 s.testDB.DB,
	}
}

func (s *NodeReportGeneratorTestSuite) TearDownSuite() {
	s.truncateTable(postgresSchema.NodeCvesTableName)
	s.truncateTable(postgresSchema.NodeComponentsCvesEdgesTableName)
	s.truncateTable(postgresSchema.NodeComponentsTableName)
	s.truncateTable(postgresSchema.NodeComponentEdgesTableName)
	s.truncateTable(postgresSchema.NodesTableName)
}

func (s *NodeReportGeneratorTestSuite) truncateTable(name string) {
	sql := fmt.Sprintf("TRUNCATE %s CASCADE", name)
	_, err := s.testDB.Exec(s.ctx, sql)
	s.NoError(err)
}

func (s *NodeReportGeneratorTestSuite) testNodes() []*storage.Node {
	t, err := protocompat.ConvertTimeToTimestampOrError(time.Unix(0, 1000))
	s.Require().NoError(err)

	var nodes []*storage.Node
	for _, cluster := range s.clusters {
		for i := range 2 {
			nodeName := fmt.Sprintf("%s_node%d", cluster.GetName(), i)
			node := &storage.Node{
				Id:          uuid.NewV4().String(),
				Name:        nodeName,
				ClusterId:   cluster.GetId(),
				ClusterName: cluster.GetName(),
				Scan: &storage.NodeScan{
					ScanTime:        protocompat.TimestampNow(),
					OperatingSystem: "linux",
					Components: []*storage.EmbeddedNodeScanComponent{
						{
							Name:    fmt.Sprintf("%s_comp", nodeName),
							Version: "1.0",
							Vulnerabilities: []*storage.NodeVulnerability{
								{
									CveBaseInfo: &storage.CVEInfo{
										Cve:       fmt.Sprintf("CVE-critical-%s_comp", nodeName),
										CreatedAt: t,
									},
									Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
									Cvss:     9.0,
									SetFixedBy: &storage.NodeVulnerability_FixedBy{
										FixedBy: "1.1",
									},
								},
								{
									CveBaseInfo: &storage.CVEInfo{
										Cve:       fmt.Sprintf("CVE-low-%s_comp", nodeName),
										CreatedAt: t,
									},
									Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
									Cvss:     2.0,
								},
							},
						},
					},
				},
			}
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func (s *NodeReportGeneratorTestSuite) TestGetReportData_AllNodes() {
	snap := s.testNodeReportSnapshot(nil, "", nil, storage.ReportStatus_ON_DEMAND)
	cveResponses, _, err := s.reportGenerator.getReportData(s.ctx, snap)
	s.NoError(err)

	collected := collectNodeReportData(cveResponses)
	s.ElementsMatch([]string{"c1", "c2"}, collected.clusterNames)
	s.ElementsMatch(
		[]string{"c1_node0", "c1_node1", "c2_node0", "c2_node1"},
		collected.nodeNames,
	)
	s.ElementsMatch(
		[]string{"c1_node0_comp", "c1_node1_comp", "c2_node0_comp", "c2_node1_comp"},
		collected.componentNames,
	)
	s.ElementsMatch(
		[]string{
			"CVE-critical-c1_node0_comp", "CVE-low-c1_node0_comp",
			"CVE-critical-c1_node1_comp", "CVE-low-c1_node1_comp",
			"CVE-critical-c2_node0_comp", "CVE-low-c2_node0_comp",
			"CVE-critical-c2_node1_comp", "CVE-low-c2_node1_comp",
		},
		collected.cveNames,
	)
}

func (s *NodeReportGeneratorTestSuite) TestGetReportData_EntityScope_SingleCluster() {
	entityScope := &storage.EntityScope{
		Rules: []*storage.EntityScopeRule{
			{
				Entity: storage.EntityType_ENTITY_TYPE_CLUSTER,
				Field:  storage.EntityField_FIELD_NAME,
				Values: []*storage.RuleValue{
					{Value: "c1", MatchType: storage.MatchType_EXACT},
				},
			},
		},
	}
	scopeRules := []*storage.SimpleAccessScope_Rules{
		{IncludedClusters: []string{"c1"}},
	}
	snap := s.testNodeReportSnapshot(entityScope, "", scopeRules, storage.ReportStatus_ON_DEMAND)
	cveResponses, _, err := s.reportGenerator.getReportData(s.ctx, snap)
	s.NoError(err)

	collected := collectNodeReportData(cveResponses)
	s.ElementsMatch([]string{"c1"}, collected.clusterNames)
	s.ElementsMatch([]string{"c1_node0", "c1_node1"}, collected.nodeNames)
	s.ElementsMatch([]string{"c1_node0_comp", "c1_node1_comp"}, collected.componentNames)
	s.ElementsMatch(
		[]string{
			"CVE-critical-c1_node0_comp", "CVE-low-c1_node0_comp",
			"CVE-critical-c1_node1_comp", "CVE-low-c1_node1_comp",
		},
		collected.cveNames,
	)
}

func (s *NodeReportGeneratorTestSuite) TestGetReportData_PartiallyIncludedCluster() {
	// User has namespace-only access (no cluster-level rule) for c1.
	// Nodes from c1 should still be visible because the cluster is partially included.
	scopeRules := []*storage.SimpleAccessScope_Rules{
		{
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
				{ClusterName: "c1", NamespaceName: "ns1"},
			},
		},
	}
	snap := s.testNodeReportSnapshot(nil, "", scopeRules, storage.ReportStatus_VIEW_BASED)
	cveResponses, _, err := s.reportGenerator.getReportData(s.ctx, snap)
	s.NoError(err)

	collected := collectNodeReportData(cveResponses)
	s.ElementsMatch([]string{"c1"}, collected.clusterNames)
	s.ElementsMatch([]string{"c1_node0", "c1_node1"}, collected.nodeNames)
}

func (s *NodeReportGeneratorTestSuite) TestGetReportData_EntityScope_WithCVSSFilter() {
	entityScope := &storage.EntityScope{
		Rules: []*storage.EntityScopeRule{
			{
				Entity: storage.EntityType_ENTITY_TYPE_CLUSTER,
				Field:  storage.EntityField_FIELD_NAME,
				Values: []*storage.RuleValue{
					{Value: "c1", MatchType: storage.MatchType_EXACT},
				},
			},
		},
	}
	scopeRules := []*storage.SimpleAccessScope_Rules{
		{IncludedClusters: []string{"c1"}},
	}
	snap := s.testNodeReportSnapshot(entityScope, "CVSS:>=7.0", scopeRules, storage.ReportStatus_ON_DEMAND)
	cveResponses, _, err := s.reportGenerator.getReportData(s.ctx, snap)
	s.NoError(err)

	collected := collectNodeReportData(cveResponses)
	s.ElementsMatch([]string{"c1"}, collected.clusterNames)
	s.ElementsMatch([]string{"c1_node0", "c1_node1"}, collected.nodeNames)
	s.ElementsMatch([]string{"c1_node0_comp", "c1_node1_comp"}, collected.componentNames)
	s.ElementsMatch(
		[]string{
			"CVE-critical-c1_node0_comp",
			"CVE-critical-c1_node1_comp",
		},
		collected.cveNames,
	)
}

func (s *NodeReportGeneratorTestSuite) TestGetReportData_ViewBased() {
	scopeRules := []*storage.SimpleAccessScope_Rules{
		{IncludedClusters: []string{"c1"}},
	}
	snap := s.testNodeReportSnapshot(nil, "CVSS:>=7.0", scopeRules, storage.ReportStatus_VIEW_BASED)
	cveResponses, _, err := s.reportGenerator.getReportData(s.ctx, snap)
	s.NoError(err)

	collected := collectNodeReportData(cveResponses)
	s.ElementsMatch([]string{"c1"}, collected.clusterNames)
	s.ElementsMatch([]string{"c1_node0", "c1_node1"}, collected.nodeNames)
	s.ElementsMatch([]string{"c1_node0_comp", "c1_node1_comp"}, collected.componentNames)
	s.ElementsMatch(
		[]string{
			"CVE-critical-c1_node0_comp",
			"CVE-critical-c1_node1_comp",
		},
		collected.cveNames,
	)
}

func (s *NodeReportGeneratorTestSuite) TestGetReportData_ClusterRegexScope() {
	entityScope := &storage.EntityScope{
		Rules: []*storage.EntityScopeRule{
			{
				Entity: storage.EntityType_ENTITY_TYPE_CLUSTER,
				Field:  storage.EntityField_FIELD_NAME,
				Values: []*storage.RuleValue{
					{Value: "c.*", MatchType: storage.MatchType_REGEX},
				},
			},
		},
	}
	snap := s.testNodeReportSnapshot(entityScope, "", nil, storage.ReportStatus_ON_DEMAND)
	cveResponses, _, err := s.reportGenerator.getReportData(s.ctx, snap)
	s.NoError(err)

	collected := collectNodeReportData(cveResponses)
	s.ElementsMatch([]string{"c1", "c2"}, collected.clusterNames)
	s.ElementsMatch(
		[]string{"c1_node0", "c1_node1", "c2_node0", "c2_node1"},
		collected.nodeNames,
	)
	s.ElementsMatch(
		[]string{"c1_node0_comp", "c1_node1_comp", "c2_node0_comp", "c2_node1_comp"},
		collected.componentNames,
	)
	s.ElementsMatch(
		[]string{
			"CVE-critical-c1_node0_comp", "CVE-low-c1_node0_comp",
			"CVE-critical-c1_node1_comp", "CVE-low-c1_node1_comp",
			"CVE-critical-c2_node0_comp", "CVE-low-c2_node0_comp",
			"CVE-critical-c2_node1_comp", "CVE-low-c2_node1_comp",
		},
		collected.cveNames,
	)
}

func (s *NodeReportGeneratorTestSuite) TestGetReportData_CancelledContext() {
	snap := s.testNodeReportSnapshot(nil, "", nil, storage.ReportStatus_ON_DEMAND)
	ctx, cancel := context.WithCancel(s.ctx)
	cancel()

	_, _, err := s.reportGenerator.getReportData(ctx, snap)
	s.Error(err)
	s.ErrorIs(err, context.Canceled)
}

func (s *NodeReportGeneratorTestSuite) testNodeReportSnapshot(
	entityScope *storage.EntityScope,
	query string,
	scopeRules []*storage.SimpleAccessScope_Rules,
	requestType storage.ReportStatus_RunMethod,
) *storage.ReportSnapshot {
	snap := fixtures.GetReportSnapshot()
	snap.Filter = &storage.ReportSnapshot_NodeVulnReportFilters{
		NodeVulnReportFilters: &storage.NodeVulnerabilityReportFilters{
			Query:            query,
			AccessScopeRules: scopeRules,
		},
	}
	snap.ReportStatus = &storage.ReportStatus{
		ReportRequestType: requestType,
	}
	if entityScope != nil {
		snap.ResourceScope = &storage.ResourceScope{
			ScopeReference: &storage.ResourceScope_EntityScope{
				EntityScope: entityScope,
			},
		}
	} else {
		snap.ResourceScope = nil
	}
	snap.Collection = nil
	return snap
}

func collectNodeReportData(responses []*NodeCVEQueryResponse) *nodeReportData {
	clusterNames := set.NewStringSet()
	nodeNames := set.NewStringSet()
	componentNames := set.NewStringSet()
	cveNames := make([]string, 0, len(responses))

	for _, res := range responses {
		if res.GetCluster() != "" {
			clusterNames.Add(res.GetCluster())
		}
		if res.GetNode() != "" {
			nodeNames.Add(res.GetNode())
		}
		if res.GetComponent() != "" {
			componentNames.Add(res.GetComponent())
		}
		cveNames = append(cveNames, res.GetCVE())
	}
	return &nodeReportData{
		clusterNames:   clusterNames.AsSlice(),
		nodeNames:      nodeNames.AsSlice(),
		componentNames: componentNames.AsSlice(),
		cveNames:       cveNames,
	}
}
