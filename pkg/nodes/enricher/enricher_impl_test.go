package enricher

import (
	"testing"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/logging"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"golang.org/x/sync/semaphore"
)

const allSkippedWarnSnippet = "No applicable node scanner ran"

// swapObservedLogger replaces the package-level logger with an observer-backed one at Warn level and
// returns the observed logs plus a restore func. Lets us assert exactly which diagnostic warnings fire.
func swapObservedLogger(t *testing.T) *observer.ObservedLogs {
	core, logs := observer.New(zap.WarnLevel)
	orig := log
	log = &logging.LoggerImpl{InnerLogger: zap.New(core).Sugar()}
	t.Cleanup(func() { log = orig })
	return logs
}

var _ types.NodeScannerWithDataSource = (*fakeNodeScannerWithDataSource)(nil)

type fakeNodeScannerWithDataSource struct {
	nodeScanner types.NodeScanner
}

type opts struct {
	requestedScan bool
	scannerType   string
}

func newFakeNodeScannerWithDataSource(opts opts) types.NodeScannerWithDataSource {
	return &fakeNodeScannerWithDataSource{
		nodeScanner: &fakeNodeScanner{
			requestedScan: opts.requestedScan,
			scannerType:   opts.scannerType,
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
	scannerType   string
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

func (f *fakeNodeScanner) GetNodeInventoryScan(*storage.Node, *storage.NodeInventory, *v4.IndexReport) (*storage.NodeScan, error) {
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

func (f *fakeNodeScanner) Type() string {
	if f.scannerType != "" {
		return f.scannerType
	}
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
				v.Suppressed = true
			}
		}

		// Data moved from Vulns to Vulnerabilities in Postgres.  So simply add the data here.
		for _, v := range c.GetVulnerabilities() {
			if v.GetCveBaseInfo().GetCve() == "CVE-2020-1234" {
				v.Snoozed = true
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
				Id:   fixtureconsts.Node1,
				Scan: &storage.NodeScan{},
			},
			fns: newFakeNodeScannerWithDataSource(opts{
				requestedScan: true,
			}),
		},
		{
			name: "node does not have scan",
			node: &storage.Node{
				Id: fixtureconsts.Node1,
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

	cases := []struct {
		name string
		node *storage.Node

		fns types.NodeScannerWithDataSource
	}{
		{
			name: "node already has scan",
			node: &storage.Node{
				Id:   fixtureconsts.Node1,
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

	node := &storage.Node{Id: fixtureconsts.Node1}
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

	node := &storage.Node{Id: fixtureconsts.Node1, ClusterName: "cluster", Name: "node"}
	err := enricherImpl.EnrichNode(node)
	assert.Error(t, err)
	expectedErrMsg := "error scanning node cluster:node error: no node scanners are integrated"
	assert.Equal(t, expectedErrMsg, err.Error())
}

// TestEnrichWithScan_AllScannersSkippedWarning validates the ROX-36432 diagnostic: when an IndexReport
// arrives but every registered scanner is inapplicable (no ScannerV4 registered), we warn - and, per
// review, only for the v4/nodeindex path (indexReport != nil), not for benign v2-on-a-v4-only-Central.
func TestEnrichWithScan_AllScannersSkippedWarning(t *testing.T) {
	node := &storage.Node{Id: fixtureconsts.Node1, ClusterName: "cluster", Name: "node"}

	newEnricher := func(scanner types.NodeScannerWithDataSource) *enricherImpl {
		return &enricherImpl{
			cves: &fakeCVESuppressor{},
			scanners: map[string]types.NodeScannerWithDataSource{
				scanner.GetNodeScanner().Type(): scanner,
			},
			metrics: newMetrics(pkgMetrics.CentralSubsystem),
		}
	}

	t.Run("warns when only v2 registered and an IndexReport arrives", func(t *testing.T) {
		logs := swapObservedLogger(t)
		clairify := newFakeNodeScannerWithDataSource(opts{scannerType: types.Clairify})
		e := newEnricher(clairify)
		testNode := node.CloneVT()

		err := e.enrichWithScan(testNode, nil, &v4.IndexReport{})
		require.NoError(t, err)
		assert.Nil(t, testNode.GetScan(), "node scan must be left untouched when every scanner is skipped")
		assert.Len(t, logs.FilterMessageSnippet(allSkippedWarnSnippet).All(), 1)
	})

	t.Run("does not warn on the v2 path (indexReport == nil)", func(t *testing.T) {
		logs := swapObservedLogger(t)
		// Only ScannerV4 registered; a v2 NodeInventory arrives - v4 is skipped, all skipped, but the gate
		// (indexReport != nil) must suppress the warning.
		v4Scanner := newFakeNodeScannerWithDataSource(opts{scannerType: types.ScannerV4})
		e := newEnricher(v4Scanner)
		testNode := node.CloneVT()

		err := e.enrichWithScan(testNode, &storage.NodeInventory{}, nil)
		require.NoError(t, err)
		assert.Empty(t, logs.FilterMessageSnippet(allSkippedWarnSnippet).All(),
			"must not warn for the routine v2-on-a-v4-only-Central case")
	})

	t.Run("does not warn when a scanner actually runs", func(t *testing.T) {
		logs := swapObservedLogger(t)
		v4Scanner := newFakeNodeScannerWithDataSource(opts{scannerType: types.ScannerV4})
		e := newEnricher(v4Scanner)
		testNode := node.CloneVT()

		err := e.enrichWithScan(testNode, nil, &v4.IndexReport{})
		require.NoError(t, err)
		assert.NotNil(t, testNode.GetScan(), "scanner should have populated the scan")
		assert.Empty(t, logs.FilterMessageSnippet(allSkippedWarnSnippet).All())
	})
}

func TestFillScanStatsWithPostgres(t *testing.T) {
	cases := []struct {
		node                 *storage.Node
		expectedVulns        int32
		expectedFixableVulns int32
	}{
		{
			node: &storage.Node{
				Id: fixtureconsts.Node1,
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
				Id: fixtureconsts.Node1,
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
				Id: fixtureconsts.Node1,
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
