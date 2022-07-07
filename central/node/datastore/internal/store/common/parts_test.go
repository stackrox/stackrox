package common

import (
	"testing"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stretchr/testify/assert"
)

func TestSplitAndMergeNode(t *testing.T) {
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
			ScanTime: ts,
			Components: []*storage.EmbeddedNodeScanComponent{
				{
					Name:    "comp1",
					Version: "ver1",
					Vulns:   []*storage.EmbeddedVulnerability{},
				},
				{
					Name:    "comp1",
					Version: "ver2",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:               "cve1",
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
						},
						{
							Cve:               "cve2",
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "ver3",
							},
						},
					},
				},
				{
					Name:    "comp2",
					Version: "ver1",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:               "cve1",
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "ver2",
							},
						},
						{
							Cve:               "cve2",
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
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
				Component: &storage.ImageComponent{
					Id:      scancomponent.ComponentID("comp1", "ver1", ""),
					Name:    "comp1",
					Version: "ver1",
					Source:  storage.SourceType_INFRASTRUCTURE,
				},
				Edge: &storage.NodeComponentEdge{
					Id:              edges.EdgeID{ParentID: "id", ChildID: scancomponent.ComponentID("comp1", "ver1", "")}.ToString(),
					NodeId:          "id",
					NodeComponentId: scancomponent.ComponentID("comp1", "ver1", ""),
				},
				Children: []*CVEParts{},
			},
			{
				Component: &storage.ImageComponent{
					Id:      scancomponent.ComponentID("comp1", "ver2", ""),
					Name:    "comp1",
					Version: "ver2",
					Source:  storage.SourceType_INFRASTRUCTURE,
				},
				Edge: &storage.NodeComponentEdge{
					Id:              edges.EdgeID{ParentID: "id", ChildID: scancomponent.ComponentID("comp1", "ver2", "")}.ToString(),
					NodeId:          "id",
					NodeComponentId: scancomponent.ComponentID("comp1", "ver2", ""),
				},
				Children: []*CVEParts{
					{
						CVE: &storage.CVE{
							Id:   "cve1",
							Type: storage.CVE_NODE_CVE,
						},
						Edge: &storage.ComponentCVEEdge{
							Id:               edges.EdgeID{ParentID: scancomponent.ComponentID("comp1", "ver2", ""), ChildID: "cve1"}.ToString(),
							ImageComponentId: scancomponent.ComponentID("comp1", "ver2", ""),
							ImageCveId:       "cve1",
						},
					},
					{
						CVE: &storage.CVE{
							Id:   "cve2",
							Type: storage.CVE_NODE_CVE,
						},
						Edge: &storage.ComponentCVEEdge{
							Id: edges.EdgeID{ParentID: scancomponent.ComponentID("comp1", "ver2", ""), ChildID: "cve2"}.ToString(),
							HasFixedBy: &storage.ComponentCVEEdge_FixedBy{
								FixedBy: "ver3",
							},
							IsFixable:        true,
							ImageComponentId: scancomponent.ComponentID("comp1", "ver2", ""),
							ImageCveId:       "cve2",
						},
					},
				},
			},
			{
				Component: &storage.ImageComponent{
					Id:      scancomponent.ComponentID("comp2", "ver1", ""),
					Name:    "comp2",
					Version: "ver1",
					Source:  storage.SourceType_INFRASTRUCTURE,
				},
				Edge: &storage.NodeComponentEdge{
					Id:              edges.EdgeID{ParentID: "id", ChildID: scancomponent.ComponentID("comp2", "ver1", "")}.ToString(),
					NodeId:          "id",
					NodeComponentId: scancomponent.ComponentID("comp2", "ver1", ""),
				},
				Children: []*CVEParts{
					{
						CVE: &storage.CVE{
							Id:   "cve1",
							Type: storage.CVE_NODE_CVE,
						},
						Edge: &storage.ComponentCVEEdge{
							Id: edges.EdgeID{ParentID: scancomponent.ComponentID("comp2", "ver1", ""), ChildID: "cve1"}.ToString(),
							HasFixedBy: &storage.ComponentCVEEdge_FixedBy{
								FixedBy: "ver2",
							},
							IsFixable:        true,
							ImageComponentId: scancomponent.ComponentID("comp2", "ver1", ""),
							ImageCveId:       "cve1",
						},
					},
					{
						CVE: &storage.CVE{
							Id:   "cve2",
							Type: storage.CVE_NODE_CVE,
						},
						Edge: &storage.ComponentCVEEdge{
							Id:               edges.EdgeID{ParentID: scancomponent.ComponentID("comp2", "ver1", ""), ChildID: "cve2"}.ToString(),
							ImageComponentId: scancomponent.ComponentID("comp2", "ver1", ""),
							ImageCveId:       "cve2",
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
