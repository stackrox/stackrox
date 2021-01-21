package enricher

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/expiringcache"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
)

type fakeNodeScanner struct {
	requestedScan bool
}

func (f *fakeNodeScanner) MaxConcurrentNodeScanSemaphore() *semaphore.Weighted {
	return semaphore.NewWeighted(1)
}

func (f *fakeNodeScanner) GetNodeScan(_ *storage.Node) (*storage.NodeScan, error) {
	f.requestedScan = true
	return &storage.NodeScan{
		Components: []*storage.EmbeddedNodeScanComponent{
			{
				Vulns: []*storage.EmbeddedVulnerability{
					{
						Cve: "CVE-2020-1234",
					},
				},
			},
		},
	}, nil
}

func (f *fakeNodeScanner) TestNodeScanner() error {
	return nil
}

func (f *fakeNodeScanner) Type() string {
	return "type"
}

func (f *fakeNodeScanner) Name() string {
	return "name"
}

func (f *fakeNodeScanner) DataSource() *storage.DataSource {
	return nil
}

type fakeCVESuppressor struct{}

func (f *fakeCVESuppressor) EnrichNodeWithSuppressedCVEs(node *storage.Node) {
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
		name                string
		ctx                 EnrichmentContext
		inScanCache         bool
		shortCircuitScanner bool
		node                *storage.Node

		fns *fakeNodeScanner
	}{
		{
			name: "nothing in the cache",
			ctx: EnrichmentContext{
				FetchOpt: UseCachesIfPossible,
			},
			inScanCache: false,
			node:        &storage.Node{Id: "id"},

			fns: &fakeNodeScanner{
				requestedScan: true,
			},
		},
		{
			name: "scan in cache",
			ctx: EnrichmentContext{
				FetchOpt: UseCachesIfPossible,
			},
			inScanCache:         true,
			shortCircuitScanner: true,
			node:                &storage.Node{Id: "id"},

			fns: &fakeNodeScanner{
				requestedScan: false,
			},
		},
		{
			name: "data in cache, but force refetch",
			ctx: EnrichmentContext{
				FetchOpt: ForceRefetch,
			},
			inScanCache: true,
			node:        &storage.Node{Id: "id"},

			fns: &fakeNodeScanner{
				requestedScan: true,
			},
		},
		{
			name: "data not in cache, but node already has scan",
			ctx: EnrichmentContext{
				FetchOpt: UseCachesIfPossible,
			},
			inScanCache:         false,
			shortCircuitScanner: true,
			node: &storage.Node{
				Id:   "id",
				Scan: &storage.NodeScan{},
			},
			fns: &fakeNodeScanner{
				requestedScan: false,
			},
		},
		{
			name: "data not in cache and ignore existing nodes",
			ctx: EnrichmentContext{
				FetchOpt: IgnoreExistingNodes,
			},
			inScanCache: false,
			node: &storage.Node{
				Id:   "id",
				Scan: &storage.NodeScan{},
			},
			fns: &fakeNodeScanner{
				requestedScan: true,
			},
		},
		{
			name: "data in cache and ignore existing nodes",
			ctx: EnrichmentContext{
				FetchOpt: IgnoreExistingNodes,
			},
			inScanCache:         true,
			shortCircuitScanner: true,
			node: &storage.Node{
				Id:   "id",
				Scan: &storage.NodeScan{},
			},
			fns: &fakeNodeScanner{
				requestedScan: false,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			fns := &fakeNodeScanner{}

			enricherImpl := &enricherImpl{
				cves: &fakeCVESuppressor{},
				scanners: map[string]types.NodeScannerWithDataSource{
					fns.Type(): fns,
				},
				scanCache: expiringcache.NewExpiringCache(1 * time.Minute),
				metrics:   newMetrics(pkgMetrics.CentralSubsystem),
			}

			if c.inScanCache {
				enricherImpl.scanCache.Add(c.node.GetId(), c.node.GetScan())
			}
			err := enricherImpl.EnrichNode(c.ctx, c.node)
			require.NoError(t, err)

			assert.Equal(t, c.fns, fns)
		})
	}
}

func TestCVESuppression(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fns := &fakeNodeScanner{}

	enricherImpl := &enricherImpl{
		cves: &fakeCVESuppressor{},
		scanners: map[string]types.NodeScannerWithDataSource{
			fns.Type(): fns,
		},
		scanCache: expiringcache.NewExpiringCache(1 * time.Minute),
		metrics:   newMetrics(pkgMetrics.CentralSubsystem),
	}

	node := &storage.Node{Id: "id"}
	err := enricherImpl.EnrichNode(EnrichmentContext{}, node)
	require.NoError(t, err)
	assert.True(t, node.Scan.Components[0].Vulns[0].Suppressed)
}

func TestZeroIntegrations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	enricherImpl := &enricherImpl{
		cves:      &fakeCVESuppressor{},
		scanners:  make(map[string]types.NodeScannerWithDataSource),
		scanCache: expiringcache.NewExpiringCache(1 * time.Minute),
		metrics:   newMetrics(pkgMetrics.CentralSubsystem),
	}

	node := &storage.Node{Id: "id", ClusterName: "cluster", Name: "node"}
	err := enricherImpl.EnrichNode(EnrichmentContext{}, node)
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
