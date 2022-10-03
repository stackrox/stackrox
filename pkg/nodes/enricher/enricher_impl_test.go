package enricher

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
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
	return &storage.NodeScan{
		Components: []*storage.EmbeddedNodeScanComponent{
			{
				Vulns: []*storage.EmbeddedVulnerability{
					{
						Cve: "CVE-2020-1234",
					},
					{
						Cve: "CVE-2021-1234",
					},
					{
						Cve: "CVE-2022-1234",
					},
				},
			},
		},
	}, nil
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
			if v.Cve == "CVE-2020-1234" {
				v.Suppressed = true
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
			node: &storage.Node{
				Id:   "id",
				Scan: &storage.NodeScan{},
			},
			fns: newFakeNodeScannerWithDataSource(opts{
				requestedScan: true,
			}),
		},
		{
			name: "node does not have scan",
			node: &storage.Node{
				Id: "id",
			},
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
	envIsolator := envisolator.NewEnvIsolator(t)
	envIsolator.Setenv(env.PostgresDatastoreEnabled.EnvVar(), "true")
	defer envIsolator.RestoreAll()

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		t.Skip("Skip postgres store tests")
		t.SkipNow()
	}

	cases := []struct {
		name string
		node *storage.Node

		fns types.NodeScannerWithDataSource
	}{
		{
			name: "node already has scan",
			node: &storage.Node{
				Id:   "id",
				Scan: &storage.NodeScan{},
			},
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

	node := &storage.Node{Id: "id"}
	err := enricherImpl.EnrichNode(node)
	require.NoError(t, err)
	assert.True(t, node.Scan.Components[0].Vulns[0].Suppressed)
}

func TestZeroIntegrations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	enricherImpl := &enricherImpl{
		cves:     &fakeCVESuppressor{},
		scanners: make(map[string]types.NodeScannerWithDataSource),
		metrics:  newMetrics(pkgMetrics.CentralSubsystem),
	}

	node := &storage.Node{Id: "id", ClusterName: "cluster", Name: "node"}
	err := enricherImpl.EnrichNode(node)
	assert.Error(t, err)
	expectedErrMsg := "error scanning node cluster:node error: no node scanners are integrated"
	assert.Equal(t, expectedErrMsg, err.Error())
}

func TestFillScanStats(t *testing.T) {
	cases := []struct {
		node                 *storage.Node
		expectedVulns        int32
		expectedFixableVulns int32
	}{
		{
			node: &storage.Node{
				Id: "node-1",
				Scan: &storage.NodeScan{
					Components: []*storage.EmbeddedNodeScanComponent{
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-1",
									SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
										FixedBy: "blah",
									},
								},
							},
						},
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-1",
								},
							},
						},
					},
				},
			},
			expectedVulns:        1,
			expectedFixableVulns: 1,
		},
		{
			node: &storage.Node{
				Id: "node-1",
				Scan: &storage.NodeScan{
					Components: []*storage.EmbeddedNodeScanComponent{
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-1",
									SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
										FixedBy: "blah",
									},
								},
							},
						},
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-2",
									SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
										FixedBy: "blah",
									},
								},
							},
						},
					},
				},
			},
			expectedVulns:        2,
			expectedFixableVulns: 2,
		},
		{
			node: &storage.Node{
				Id: "node-1",
				Scan: &storage.NodeScan{
					Components: []*storage.EmbeddedNodeScanComponent{
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-1",
								},
							},
						},
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-2",
								},
							},
						},
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-3",
								},
							},
						},
					},
				},
			},
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

func TestFillScanStatsWithPostgres(t *testing.T) {
	envIsolator := envisolator.NewEnvIsolator(t)
	envIsolator.Setenv(env.PostgresDatastoreEnabled.EnvVar(), "true")
	defer envIsolator.RestoreAll()

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		t.Skip("Skip postgres store tests")
		t.SkipNow()
	}

	cases := []struct {
		node                 *storage.Node
		expectedVulns        int32
		expectedFixableVulns int32
	}{
		{
			node: &storage.Node{
				Id: "node-1",
				Scan: &storage.NodeScan{
					Components: []*storage.EmbeddedNodeScanComponent{
						{
							Vulnerabilities: []*storage.NodeVulnerability{
								{
									CveBaseInfo: &storage.CVEInfo{
										Cve: "cve-1",
									},
									SetFixedBy: &storage.NodeVulnerability_FixedBy{
										FixedBy: "blah",
									},
								},
							},
						},
					},
				},
			},
			expectedVulns:        1,
			expectedFixableVulns: 1,
		},
		{
			node: &storage.Node{
				Id: "node-1",
				Scan: &storage.NodeScan{
					Components: []*storage.EmbeddedNodeScanComponent{
						{
							Vulnerabilities: []*storage.NodeVulnerability{
								{
									CveBaseInfo: &storage.CVEInfo{
										Cve: "cve-1",
									},
									SetFixedBy: &storage.NodeVulnerability_FixedBy{
										FixedBy: "blah",
									},
								},
							},
						},
						{
							Vulnerabilities: []*storage.NodeVulnerability{
								{
									CveBaseInfo: &storage.CVEInfo{
										Cve: "cve-2",
									},
									SetFixedBy: &storage.NodeVulnerability_FixedBy{
										FixedBy: "blah",
									},
								},
							},
						},
					},
				},
			},
			expectedVulns:        2,
			expectedFixableVulns: 2,
		},
		{
			node: &storage.Node{
				Id: "node-1",
				Scan: &storage.NodeScan{
					Components: []*storage.EmbeddedNodeScanComponent{
						{
							Vulnerabilities: []*storage.NodeVulnerability{
								{
									CveBaseInfo: &storage.CVEInfo{
										Cve: "cve-1",
									},
								},
							},
						},
						{
							Vulnerabilities: []*storage.NodeVulnerability{
								{
									CveBaseInfo: &storage.CVEInfo{
										Cve: "cve-2",
									},
								},
							},
						},
						{
							Vulnerabilities: []*storage.NodeVulnerability{
								{
									CveBaseInfo: &storage.CVEInfo{
										Cve: "cve-3",
									},
								},
							},
						},
					},
				},
			},
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
