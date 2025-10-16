package common

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/scancomponent"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestSplitAndMergeNode(t *testing.T) {
	ts := protocompat.TimestampNow()
	node := storage.Node_builder{
		Id:          "id",
		Name:        "name",
		Components:  proto.Int32(3),
		Cves:        proto.Int32(4),
		FixableCves: proto.Int32(2),
		Scan: storage.NodeScan_builder{
			ScanTime:        ts,
			OperatingSystem: "",
			Components: []*storage.EmbeddedNodeScanComponent{
				storage.EmbeddedNodeScanComponent_builder{
					Name:            "comp1",
					Version:         "ver1",
					Vulnerabilities: []*storage.NodeVulnerability{},
				}.Build(),
				storage.EmbeddedNodeScanComponent_builder{
					Name:    "comp1",
					Version: "ver2",
					Vulnerabilities: []*storage.NodeVulnerability{
						storage.NodeVulnerability_builder{
							CveBaseInfo: storage.CVEInfo_builder{
								Cve: "cve1",
							}.Build(),
						}.Build(),
						storage.NodeVulnerability_builder{
							CveBaseInfo: storage.CVEInfo_builder{
								Cve: "cve2",
							}.Build(),
							FixedBy: proto.String("ver3"),
						}.Build(),
					},
				}.Build(),
				storage.EmbeddedNodeScanComponent_builder{
					Name:    "comp2",
					Version: "ver1",
					Vulnerabilities: []*storage.NodeVulnerability{
						storage.NodeVulnerability_builder{
							CveBaseInfo: storage.CVEInfo_builder{
								Cve: "cve1",
							}.Build(),
							FixedBy: proto.String("ver2"),
						}.Build(),
						storage.NodeVulnerability_builder{
							CveBaseInfo: storage.CVEInfo_builder{
								Cve: "cve2",
							}.Build(),
						}.Build(),
					},
				}.Build(),
			},
		}.Build(),
	}.Build()

	splitExpected := &NodeParts{
		Node: storage.Node_builder{
			Id:   "id",
			Name: "name",
			Scan: storage.NodeScan_builder{
				ScanTime: ts,
			}.Build(),
			Components:  proto.Int32(3),
			Cves:        proto.Int32(4),
			FixableCves: proto.Int32(2),
		}.Build(),
		Children: []*ComponentParts{
			{
				Component: storage.NodeComponent_builder{
					Id:      scancomponent.ComponentID("comp1", "ver1", ""),
					Name:    "comp1",
					Version: "ver1",
				}.Build(),
				Edge: storage.NodeComponentEdge_builder{
					Id:              pgSearch.IDFromPks([]string{"id", scancomponent.ComponentID("comp1", "ver1", "")}),
					NodeId:          "id",
					NodeComponentId: scancomponent.ComponentID("comp1", "ver1", ""),
				}.Build(),
				Children: []*CVEParts{},
			},
			{
				Component: storage.NodeComponent_builder{
					Id:      scancomponent.ComponentID("comp1", "ver2", ""),
					Name:    "comp1",
					Version: "ver2",
				}.Build(),
				Edge: storage.NodeComponentEdge_builder{
					Id:              pgSearch.IDFromPks([]string{"id", scancomponent.ComponentID("comp1", "ver2", "")}),
					NodeId:          "id",
					NodeComponentId: scancomponent.ComponentID("comp1", "ver2", ""),
				}.Build(),
				Children: []*CVEParts{
					{
						CVE: storage.NodeCVE_builder{
							Id: cve.ID("cve1", ""),
							CveBaseInfo: storage.CVEInfo_builder{
								Cve: "cve1",
							}.Build(),
						}.Build(),
						Edge: storage.NodeComponentCVEEdge_builder{
							Id:              pgSearch.IDFromPks([]string{scancomponent.ComponentID("comp1", "ver2", ""), cve.ID("cve1", "")}),
							NodeComponentId: scancomponent.ComponentID("comp1", "ver2", ""),
							NodeCveId:       cve.ID("cve1", ""),
						}.Build(),
					},
					{
						CVE: storage.NodeCVE_builder{
							Id: cve.ID("cve2", ""),
							CveBaseInfo: storage.CVEInfo_builder{
								Cve: "cve2",
							}.Build(),
						}.Build(),
						Edge: storage.NodeComponentCVEEdge_builder{
							Id:              pgSearch.IDFromPks([]string{scancomponent.ComponentID("comp1", "ver2", ""), cve.ID("cve2", "")}),
							FixedBy:         proto.String("ver3"),
							IsFixable:       true,
							NodeComponentId: scancomponent.ComponentID("comp1", "ver2", ""),
							NodeCveId:       cve.ID("cve2", ""),
						}.Build(),
					},
				},
			},
			{
				Component: storage.NodeComponent_builder{
					Id:      scancomponent.ComponentID("comp2", "ver1", ""),
					Name:    "comp2",
					Version: "ver1",
				}.Build(),
				Edge: storage.NodeComponentEdge_builder{
					Id:              pgSearch.IDFromPks([]string{"id", scancomponent.ComponentID("comp2", "ver1", "")}),
					NodeId:          "id",
					NodeComponentId: scancomponent.ComponentID("comp2", "ver1", ""),
				}.Build(),
				Children: []*CVEParts{
					{
						CVE: storage.NodeCVE_builder{
							Id: cve.ID("cve1", ""),
							CveBaseInfo: storage.CVEInfo_builder{
								Cve: "cve1",
							}.Build(),
						}.Build(),
						Edge: storage.NodeComponentCVEEdge_builder{
							Id:              pgSearch.IDFromPks([]string{scancomponent.ComponentID("comp2", "ver1", ""), cve.ID("cve1", "")}),
							FixedBy:         proto.String("ver2"),
							IsFixable:       true,
							NodeComponentId: scancomponent.ComponentID("comp2", "ver1", ""),
							NodeCveId:       cve.ID("cve1", ""),
						}.Build(),
					},
					{
						CVE: storage.NodeCVE_builder{
							Id: cve.ID("cve2", ""),
							CveBaseInfo: storage.CVEInfo_builder{
								Cve: "cve2",
							}.Build(),
						}.Build(),
						Edge: storage.NodeComponentCVEEdge_builder{
							Id:              pgSearch.IDFromPks([]string{scancomponent.ComponentID("comp2", "ver1", ""), cve.ID("cve2", "")}),
							NodeComponentId: scancomponent.ComponentID("comp2", "ver1", ""),
							NodeCveId:       cve.ID("cve2", ""),
						}.Build(),
					},
				},
			},
		},
	}

	splitActual := Split(node, true)
	protoassert.Equal(t, splitExpected.Node, splitActual.Node)

	assert.Len(t, splitActual.Children, len(splitExpected.Children))
	for i, expected := range splitExpected.Children {
		actual := splitActual.Children[i]
		protoassert.Equal(t, expected.Component, actual.Component)
		protoassert.Equal(t, expected.Edge, actual.Edge)

		assert.Len(t, actual.Children, len(expected.Children))
		for i, e := range expected.Children {
			a := actual.Children[i]
			protoassert.Equal(t, e.Edge, a.Edge)
			protoassert.Equal(t, e.CVE, a.CVE)
		}
	}

	nodeActual := Merge(splitActual)
	protoassert.Equal(t, node, nodeActual)
}
