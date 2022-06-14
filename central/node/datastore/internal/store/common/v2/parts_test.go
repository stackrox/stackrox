package common

import (
	"testing"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/cve"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/scancomponent"
	"github.com/stackrox/stackrox/pkg/search/postgres"
	"github.com/stackrox/stackrox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/assert"
)

func TestSplitAndMergeNode(t *testing.T) {
	envIsolator := envisolator.NewEnvIsolator(t)
	envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")
	defer envIsolator.RestoreAll()

	if !features.PostgresDatastore.Enabled() {
		t.Skip("Skip postgres tests")
		t.SkipNow()
	}

	ts := timestamp.TimestampNow()
	node := &storage.Node{
		Id:   "id",
		Name: "name",
		SetComponents: &storage.Node_Components{
			Components: 3,
		},
		SetCves: &storage.Node_Cves{
			Cves: 4,
		},
		SetFixable: &storage.Node_FixableCves{
			FixableCves: 2,
		},
		Scan: &storage.NodeScan{
			ScanTime:        ts,
			OperatingSystem: "",
			Components: []*storage.EmbeddedNodeScanComponent{
				{
					Name:            "comp1",
					Version:         "ver1",
					Vulnerabilities: []*storage.NodeVulnerability{},
				},
				{
					Name:    "comp1",
					Version: "ver2",
					Vulnerabilities: []*storage.NodeVulnerability{
						{
							CveBaseInfo: &storage.CVEInfo{
								Cve: "cve1",
							},
						},
						{
							CveBaseInfo: &storage.CVEInfo{
								Cve: "cve2",
							},
							SetFixedBy: &storage.NodeVulnerability_FixedBy{
								FixedBy: "ver3",
							},
						},
					},
				},
				{
					Name:    "comp2",
					Version: "ver1",
					Vulnerabilities: []*storage.NodeVulnerability{
						{
							CveBaseInfo: &storage.CVEInfo{
								Cve: "cve1",
							},
							SetFixedBy: &storage.NodeVulnerability_FixedBy{
								FixedBy: "ver2",
							},
						},
						{
							CveBaseInfo: &storage.CVEInfo{
								Cve: "cve2",
							},
						},
					},
				},
			},
		},
	}

	splitExpected := &NodeParts{
		Node: &storage.Node{
			Id:   "id",
			Name: "name",
			Scan: &storage.NodeScan{
				ScanTime: ts,
			},
			SetComponents: &storage.Node_Components{
				Components: 3,
			},
			SetCves: &storage.Node_Cves{
				Cves: 4,
			},
			SetFixable: &storage.Node_FixableCves{
				FixableCves: 2,
			},
		},
		Children: []*ComponentParts{
			{
				Component: &storage.NodeComponent{
					Id:      scancomponent.ComponentID("comp1", "ver1", ""),
					Name:    "comp1",
					Version: "ver1",
				},
				Edge: &storage.NodeComponentEdge{
					Id:              postgres.IDFromPks([]string{"id", scancomponent.ComponentID("comp1", "ver1", "")}),
					NodeId:          "id",
					NodeComponentId: scancomponent.ComponentID("comp1", "ver1", ""),
				},
				Children: []*CVEParts{},
			},
			{
				Component: &storage.NodeComponent{
					Id:      scancomponent.ComponentID("comp1", "ver2", ""),
					Name:    "comp1",
					Version: "ver2",
				},
				Edge: &storage.NodeComponentEdge{
					Id:              postgres.IDFromPks([]string{"id", scancomponent.ComponentID("comp1", "ver2", "")}),
					NodeId:          "id",
					NodeComponentId: scancomponent.ComponentID("comp1", "ver2", ""),
				},
				Children: []*CVEParts{
					{
						CVE: &storage.NodeCVE{
							Id: cve.ID("cve1", ""),
							CveBaseInfo: &storage.CVEInfo{
								Cve: "cve1",
							},
						},
						Edge: &storage.NodeComponentCVEEdge{
							Id:              postgres.IDFromPks([]string{scancomponent.ComponentID("comp1", "ver2", ""), cve.ID("cve1", "")}),
							NodeComponentId: scancomponent.ComponentID("comp1", "ver2", ""),
							NodeCveId:       cve.ID("cve1", ""),
						},
					},
					{
						CVE: &storage.NodeCVE{
							Id: cve.ID("cve2", ""),
							CveBaseInfo: &storage.CVEInfo{
								Cve: "cve2",
							},
						},
						Edge: &storage.NodeComponentCVEEdge{
							Id: postgres.IDFromPks([]string{scancomponent.ComponentID("comp1", "ver2", ""), cve.ID("cve2", "")}),
							HasFixedBy: &storage.NodeComponentCVEEdge_FixedBy{
								FixedBy: "ver3",
							},
							IsFixable:       true,
							NodeComponentId: scancomponent.ComponentID("comp1", "ver2", ""),
							NodeCveId:       cve.ID("cve2", ""),
						},
					},
				},
			},
			{
				Component: &storage.NodeComponent{
					Id:      scancomponent.ComponentID("comp2", "ver1", ""),
					Name:    "comp2",
					Version: "ver1",
				},
				Edge: &storage.NodeComponentEdge{
					Id:              postgres.IDFromPks([]string{"id", scancomponent.ComponentID("comp2", "ver1", "")}),
					NodeId:          "id",
					NodeComponentId: scancomponent.ComponentID("comp2", "ver1", ""),
				},
				Children: []*CVEParts{
					{
						CVE: &storage.NodeCVE{
							Id: cve.ID("cve1", ""),
							CveBaseInfo: &storage.CVEInfo{
								Cve: "cve1",
							},
						},
						Edge: &storage.NodeComponentCVEEdge{
							Id: postgres.IDFromPks([]string{scancomponent.ComponentID("comp2", "ver1", ""), cve.ID("cve1", "")}),
							HasFixedBy: &storage.NodeComponentCVEEdge_FixedBy{
								FixedBy: "ver2",
							},
							IsFixable:       true,
							NodeComponentId: scancomponent.ComponentID("comp2", "ver1", ""),
							NodeCveId:       cve.ID("cve1", ""),
						},
					},
					{
						CVE: &storage.NodeCVE{
							Id: cve.ID("cve2", ""),
							CveBaseInfo: &storage.CVEInfo{
								Cve: "cve2",
							},
						},
						Edge: &storage.NodeComponentCVEEdge{
							Id:              postgres.IDFromPks([]string{scancomponent.ComponentID("comp2", "ver1", ""), cve.ID("cve2", "")}),
							NodeComponentId: scancomponent.ComponentID("comp2", "ver1", ""),
							NodeCveId:       cve.ID("cve2", ""),
						},
					},
				},
			},
		},
	}

	splitActual := Split(node, true)
	assert.Equal(t, splitExpected, splitActual)

	nodeActual := Merge(splitActual)
	assert.Equal(t, node, nodeActual)
}
