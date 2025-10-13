package resolvers

import (
	"context"
	"testing"

	"github.com/graph-gophers/graphql-go"
	nodeCVEsDSMocks "github.com/stackrox/rox/central/cve/node/datastore/mocks"
	nodeDSMocks "github.com/stackrox/rox/central/node/datastore/mocks"
	nodeComponentsDSMocks "github.com/stackrox/rox/central/nodecomponent/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	nodeConverter "github.com/stackrox/rox/pkg/nodes/converter"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	nodeWithScanQuery = `
		query getNodes($query: String, $pagination: Pagination) {
			nodes(query: $query, pagination: $pagination) { 
				id
				scan {
					nodeComponents {
						name
						nodeVulnerabilities {
							cve
						}
					}
				}
			}}`

	nodeWithoutScanQuery = `
		query getNodes($query: String, $pagination: Pagination) {
			nodes(query: $query, pagination: $pagination) { 
				id
				nodeComponents {
					name
					nodeVulnerabilities {
						cve
					}
				}
			}}`
)

func TestNodeScanResolver(t *testing.T) {
	suite.Run(t, new(NodeScanResolverTestSuite))
}

type NodeScanResolverTestSuite struct {
	suite.Suite

	ctx      context.Context
	mockCtrl *gomock.Controller

	nodeDataStore          *nodeDSMocks.MockDataStore
	nodeComponentDataStore *nodeComponentsDSMocks.MockDataStore
	nodeCVEDataStore       *nodeCVEsDSMocks.MockDataStore

	resolver *Resolver
	schema   *graphql.Schema
}

func (s *NodeScanResolverTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.ctx = contextWithNodePerm(s.T(), s.mockCtrl)

	s.nodeDataStore = nodeDSMocks.NewMockDataStore(s.mockCtrl)
	s.nodeComponentDataStore = nodeComponentsDSMocks.NewMockDataStore(s.mockCtrl)
	s.nodeCVEDataStore = nodeCVEsDSMocks.NewMockDataStore(s.mockCtrl)

	s.resolver, s.schema = SetupTestResolver(s.T(), s.nodeDataStore, s.nodeComponentDataStore, s.nodeCVEDataStore, nil)
}

func (s *NodeScanResolverTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *NodeScanResolverTestSuite) TestGetNodesWithScan() {
	// Verify that full node is fetched.
	node := fixtures.GetNodeWithUniqueComponents(5, 5)
	nodeConverter.MoveNodeVulnsToNewField(node)
	s.nodeDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).
		Return([]search.Result{{
			ID: node.GetId(),
		}}, nil)
	cloned := node.CloneVT()
	cloned.Scan.Components = nil
	s.nodeDataStore.EXPECT().GetManyNodeMetadata(gomock.Any(), gomock.Any()).
		Return([]*storage.Node{cloned}, nil)
	s.nodeDataStore.EXPECT().GetNodesBatch(gomock.Any(), gomock.Any()).
		Return([]*storage.Node{node}, nil)
	response := s.schema.Exec(s.ctx, nodeWithScanQuery, "getNodes", nil)
	s.Len(response.Errors, 0)
}

func (s *NodeScanResolverTestSuite) TestGetNodesWithoutScan() {
	// Verify that full node is not fetched but rather node component and vuln stores are queried.
	node := fixtures.GetNodeWithUniqueComponents(5, 5)
	nodeConverter.MoveNodeVulnsToNewField(node)
	s.nodeDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).
		Return([]search.Result{{
			ID: node.GetId(),
		}}, nil)

	cloned := node.CloneVT()
	cloned.Scan.Components = nil
	s.nodeDataStore.EXPECT().GetManyNodeMetadata(gomock.Any(), gomock.Any()).
		Return([]*storage.Node{cloned}, nil)
	s.nodeComponentDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).
		Return(nil, nil)
	response := s.schema.Exec(s.ctx, nodeWithoutScanQuery, "getNodes", nil)
	s.Len(response.Errors, 0)
}
