package enricher

import (
	"testing"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/proto"
)

var _ types.NodeScannerWithDataSource = (*fakeNodeScannerWithDataSource)(nil)

type fakeNodeScannerWithDataSource struct {
	nodeScanner types.NodeScanner
}

type opts struct {
	requestedScan bool
}

func newFakeNodeScannerWithDataSource(opts opts) types.NodeScannerWithDataSource {
	return &fakeNodeScannerWithDataSource{
		nodeScanner: &fakeNodeScanner{
			requestedScan: opts.requestedScan,
		},
	}
}

func (f *fakeNodeScannerWithDataSource) GetNodeScanner() types.NodeScanner {
	return f.nodeScanner
}

func (*fakeNodeScannerWithDataSource) DataSource() *storage.DataSource {
	return nil
}

var _ types.NodeScanner = (*fakeNodeScanner)(nil)

type fakeNodeScanner struct {
	requestedScan bool
}

func (*fakeNodeScanner) MaxConcurrentNodeScanSemaphore() *semaphore.Weighted {
	return semaphore.NewWeighted(1)
}

func (f *fakeNodeScanner) GetNodeScan(*storage.Node) (*storage.NodeScan, error) {
	f.requestedScan = true
	return storage.NodeScan_builder{
		Components: []*storage.EmbeddedNodeScanComponent{
			storage.EmbeddedNodeScanComponent_builder{
				Vulns: []*storage.EmbeddedVulnerability{
					storage.EmbeddedVulnerability_builder{
						Cve: "CVE-2020-1234",
					}.Build(),
					storage.EmbeddedVulnerability_builder{
						Cve: "CVE-2021-1234",
					}.Build(),
					storage.EmbeddedVulnerability_builder{
						Cve: "CVE-2022-1234",
					}.Build(),
				},
			}.Build(),
		},
	}.Build(), nil
}

func (f *fakeNodeScanner) GetNodeInventoryScan(*storage.Node, *storage.NodeInventory, *v4.IndexReport) (*storage.NodeScan, error) {
	f.requestedScan = true
	return storage.NodeScan_builder{
		Components: []*storage.EmbeddedNodeScanComponent{
			storage.EmbeddedNodeScanComponent_builder{
				Vulns: []*storage.EmbeddedVulnerability{
					storage.EmbeddedVulnerability_builder{
						Cve: "CVE-2020-1234",
					}.Build(),
					storage.EmbeddedVulnerability_builder{
						Cve: "CVE-2021-1234",
					}.Build(),
					storage.EmbeddedVulnerability_builder{
						Cve: "CVE-2022-1234",
					}.Build(),
				},
			}.Build(),
		},
	}.Build(), nil
}

func (*fakeNodeScanner) TestNodeScanner() error {
	return nil
}

func (*fakeNodeScanner) Type() string {
	return "type"
}

func (*fakeNodeScanner) Name() string {
	return "name"
}

type fakeCVESuppressor struct{}

func (*fakeCVESuppressor) EnrichNodeWithSuppressedCVEs(node *storage.Node) {
	for _, c := range node.GetScan().GetComponents() {
		for _, v := range c.GetVulns() {
			if v.GetCve() == "CVE-2020-1234" {
				v.SetSuppressed(true)
			}
		}

		// Data moved from Vulns to Vulnerabilities in Postgres.  So simply add the data here.
		for _, v := range c.GetVulnerabilities() {
			if v.GetCveBaseInfo().GetCve() == "CVE-2020-1234" {
				v.SetSnoozed(true)
			}
		}
	}
}

func TestEnricherFlow(t *testing.T) {
	cases := []struct {
		name string
		node *storage.Node

		fns types.NodeScannerWithDataSource
	}{
		{
			name: "node already has scan",
			node: storage.Node_builder{
				Id:   fixtureconsts.Node1,
				Scan: &storage.NodeScan{},
			}.Build(),
			fns: newFakeNodeScannerWithDataSource(opts{
				requestedScan: true,
			}),
		},
		{
			name: "node does not have scan",
			node: storage.Node_builder{
				Id: fixtureconsts.Node1,
			}.Build(),
			fns: newFakeNodeScannerWithDataSource(opts{
				requestedScan: true,
			}),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			fns := newFakeNodeScannerWithDataSource(opts{})

			enricherImpl := &enricherImpl{
				cves: &fakeCVESuppressor{},
				scanners: map[string]types.NodeScannerWithDataSource{
					fns.GetNodeScanner().Type(): fns,
				},
				metrics: newMetrics(pkgMetrics.CentralSubsystem),
			}

			err := enricherImpl.EnrichNode(c.node)
			require.NoError(t, err)

			assert.Equal(t, c.fns, fns)
		})
	}
}

func TestEnricherFlowWithPostgres(t *testing.T) {

	cases := []struct {
		name string
		node *storage.Node

		fns types.NodeScannerWithDataSource
	}{
		{
			name: "node already has scan",
			node: storage.Node_builder{
				Id:   fixtureconsts.Node1,
				Scan: &storage.NodeScan{},
			}.Build(),
			fns: newFakeNodeScannerWithDataSource(opts{
				requestedScan: true,
			}),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			fns := newFakeNodeScannerWithDataSource(opts{})

			enricherImpl := &enricherImpl{
				cves: &fakeCVESuppressor{},
				scanners: map[string]types.NodeScannerWithDataSource{
					fns.GetNodeScanner().Type(): fns,
				},
				metrics: newMetrics(pkgMetrics.CentralSubsystem),
			}

			err := enricherImpl.EnrichNode(c.node)
			require.NoError(t, err)

			for _, c := range c.node.GetScan().GetComponents() {
				// `vulnerabilities` is the new field.
				assert.NotNil(t, c.GetVulnerabilities())
			}
		})
	}
}

func TestCVESuppression(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fns := newFakeNodeScannerWithDataSource(opts{})

	enricherImpl := &enricherImpl{
		cves: &fakeCVESuppressor{},
		scanners: map[string]types.NodeScannerWithDataSource{
			fns.GetNodeScanner().Type(): fns,
		},
		metrics: newMetrics(pkgMetrics.CentralSubsystem),
	}

	node := &storage.Node{}
	node.SetId(fixtureconsts.Node1)
	err := enricherImpl.EnrichNode(node)
	require.NoError(t, err)

	for _, c := range node.GetScan().GetComponents() {
		// `vulnerabilities` is the new field.
		assert.NotNil(t, c.GetVulnerabilities()[0].GetSnoozed())
	}
}

func TestZeroIntegrations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	enricherImpl := &enricherImpl{
		cves:     &fakeCVESuppressor{},
		scanners: make(map[string]types.NodeScannerWithDataSource),
		metrics:  newMetrics(pkgMetrics.CentralSubsystem),
	}

	node := &storage.Node{}
	node.SetId(fixtureconsts.Node1)
	node.SetClusterName("cluster")
	node.SetName("node")
	err := enricherImpl.EnrichNode(node)
	assert.Error(t, err)
	expectedErrMsg := "error scanning node cluster:node error: no node scanners are integrated"
	assert.Equal(t, expectedErrMsg, err.Error())
}

func TestFillScanStatsWithPostgres(t *testing.T) {
	cases := []struct {
		node                 *storage.Node
		expectedVulns        int32
		expectedFixableVulns int32
	}{
		{
			node: storage.Node_builder{
				Id: fixtureconsts.Node1,
				Scan: storage.NodeScan_builder{
					Components: []*storage.EmbeddedNodeScanComponent{
						storage.EmbeddedNodeScanComponent_builder{
							Vulnerabilities: []*storage.NodeVulnerability{
								storage.NodeVulnerability_builder{
									CveBaseInfo: storage.CVEInfo_builder{
										Cve: "cve-1",
									}.Build(),
									FixedBy: proto.String("blah"),
								}.Build(),
							},
						}.Build(),
					},
				}.Build(),
			}.Build(),
			expectedVulns:        1,
			expectedFixableVulns: 1,
		},
		{
			node: storage.Node_builder{
				Id: fixtureconsts.Node1,
				Scan: storage.NodeScan_builder{
					Components: []*storage.EmbeddedNodeScanComponent{
						storage.EmbeddedNodeScanComponent_builder{
							Vulnerabilities: []*storage.NodeVulnerability{
								storage.NodeVulnerability_builder{
									CveBaseInfo: storage.CVEInfo_builder{
										Cve: "cve-1",
									}.Build(),
									FixedBy: proto.String("blah"),
								}.Build(),
							},
						}.Build(),
						storage.EmbeddedNodeScanComponent_builder{
							Vulnerabilities: []*storage.NodeVulnerability{
								storage.NodeVulnerability_builder{
									CveBaseInfo: storage.CVEInfo_builder{
										Cve: "cve-2",
									}.Build(),
									FixedBy: proto.String("blah"),
								}.Build(),
							},
						}.Build(),
					},
				}.Build(),
			}.Build(),
			expectedVulns:        2,
			expectedFixableVulns: 2,
		},
		{
			node: storage.Node_builder{
				Id: fixtureconsts.Node1,
				Scan: storage.NodeScan_builder{
					Components: []*storage.EmbeddedNodeScanComponent{
						storage.EmbeddedNodeScanComponent_builder{
							Vulnerabilities: []*storage.NodeVulnerability{
								storage.NodeVulnerability_builder{
									CveBaseInfo: storage.CVEInfo_builder{
										Cve: "cve-1",
									}.Build(),
								}.Build(),
							},
						}.Build(),
						storage.EmbeddedNodeScanComponent_builder{
							Vulnerabilities: []*storage.NodeVulnerability{
								storage.NodeVulnerability_builder{
									CveBaseInfo: storage.CVEInfo_builder{
										Cve: "cve-2",
									}.Build(),
								}.Build(),
							},
						}.Build(),
						storage.EmbeddedNodeScanComponent_builder{
							Vulnerabilities: []*storage.NodeVulnerability{
								storage.NodeVulnerability_builder{
									CveBaseInfo: storage.CVEInfo_builder{
										Cve: "cve-3",
									}.Build(),
								}.Build(),
							},
						}.Build(),
					},
				}.Build(),
			}.Build(),
			expectedVulns:        3,
			expectedFixableVulns: 0,
		},
	}

	for _, c := range cases {
		t.Run(t.Name(), func(t *testing.T) {
			FillScanStats(c.node)
			assert.Equal(t, c.expectedVulns, c.node.GetCves())
			assert.Equal(t, c.expectedFixableVulns, c.node.GetFixableCves())
		})
	}
}
