package testutils

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	log = logging.LoggerForModule()
)

// DebugDeployment is a stripped down deployment to troubleshoot graph DB
// linking issues.
type DebugDeployment struct {
	ID          string
	Name        string
	Namespace   string
	ClusterID   string
	ClusterName string
}

func (d *DebugDeployment) log(offset int) {
	log.Infof(
		"%s* deployment ID %s {Name: %q, ClusterID: %q, ClusterName: %q, Namespace %q}",
		getIndent(offset),
		d.ID,
		d.Name,
		d.ClusterID,
		d.ClusterName,
		d.Namespace,
	)
}

// DebugContainer is a stripped down deployment container to troubleshoot
// graph DB linking issues.
type DebugContainer struct {
	DeploymentID string
	Idx          int
	ImageID      string
}

func (c *DebugContainer) log(offset int) {
	log.Infof(
		"%s* container {Deployment ID %q, Index %d, Image ID %q}",
		getIndent(offset),
		c.DeploymentID,
		c.Idx,
		c.ImageID,
	)
}

// DebugImage is a stripped down image to troubleshoot graph DB linking issues.
type DebugImage struct {
	ID string
}

func (i *DebugImage) log(offset int) {
	log.Infof(
		"%s* image {ID: %q}",
		getIndent(offset),
		i.ID,
	)
}

// DebugImageComponentEdge is a stripped down image to component edge to
// troubleshoot graph DB linking issues.
type DebugImageComponentEdge struct {
	ID               string
	ImageID          string
	ImageComponentID string
}

func (ice *DebugImageComponentEdge) log(offset int) {
	log.Infof(
		"%s* image component edge {ID: %q, Image ID: %q, Component ID: %q}",
		getIndent(offset),
		ice.ID,
		ice.ImageID,
		ice.ImageComponentID,
	)
}

// DebugImageComponent is a stripped down image component to troubleshoot
// graph DB linking issues.
type DebugImageComponent struct {
	ID string
}

func (ic *DebugImageComponent) log(offset int) {
	log.Infof(
		"%s* image component {ID: %q}",
		getIndent(offset),
		ic.ID,
	)
}

// DebugImageComponentCVEEdge is a stripped down image component to cve edge
// to troubleshoot graph DB linking issues.
type DebugImageComponentCVEEdge struct {
	ID               string
	ImageComponentID string
	ImageCVEID       string
}

func (cce *DebugImageComponentCVEEdge) log(offset int) {
	log.Infof(
		"%s* image component to CVE edge {ID: %q, Image Component ID: %q, CVE ID: %q}",
		getIndent(offset),
		cce.ID,
		cce.ImageComponentID,
		cce.ImageCVEID,
	)
}

// DebugImageCVEEdge is a stripped down image to CVE edge to troubleshoot
// graph DB linking issues.
type DebugImageCVEEdge struct {
	ID         string
	ImageID    string
	ImageCVEID string
	State      int32
}

func (ice *DebugImageCVEEdge) log(offset int) {
	log.Infof(
		"%s* image to CVE edge {ID: %q, Image ID: %q, CVE ID: %q, State: %d}",
		getIndent(offset),
		ice.ID,
		ice.ImageID,
		ice.ImageCVEID,
		ice.State,
	)
}

// DebugImageCVE is a stripped down image CVE to troubleshoot graph DB linking
// issues.
type DebugImageCVE struct {
	ID string
}

func (ic *DebugImageCVE) log(offset int) {
	log.Infof(
		"%s* image CVE {ID: %q}",
		getIndent(offset),
		ic.ID,
	)
}

// DebugImageGraph is a container for object lists to troubleshoot graph DB
// linking issues.
type DebugImageGraph struct {
	Deployments            []DebugDeployment
	Containers             []DebugContainer
	Images                 []DebugImage
	ImageComponentEdges    []DebugImageComponentEdge
	ImageComponents        []DebugImageComponent
	ImageComponentCVEEdges []DebugImageComponentCVEEdge
	ImageCVEEdges          []DebugImageCVEEdge
	ImageCVEs              []DebugImageCVE
}

// Log writes the collected graph data in the logs.
func (g *DebugImageGraph) Log() {
	log.Info("Image graph:")
	log.Infof("- %d deployments:", len(g.Deployments))
	for _, d := range g.Deployments {
		d.log(1)
	}
	log.Infof("- %d containers", len(g.Containers))
	for _, c := range g.Containers {
		c.log(1)
	}
	log.Infof("- %d images", len(g.Images))
	for _, i := range g.Images {
		i.log(1)
	}
	log.Infof("- %d image to component edges", len(g.ImageComponentEdges))
	for _, ice := range g.ImageComponentEdges {
		ice.log(1)
	}
	log.Infof("- %d image components", len(g.ImageComponents))
	for _, ic := range g.ImageComponents {
		ic.log(1)
	}
	log.Infof("- %d image component to CVE edges", len(g.ImageComponentCVEEdges))
	for _, cce := range g.ImageComponentCVEEdges {
		cce.log(1)
	}
	log.Infof("- %d image to CVE edges", len(g.ImageCVEEdges))
	for _, ice := range g.ImageCVEEdges {
		ice.log(1)
	}
	log.Infof("- %d image CVEs", len(g.ImageCVEs))
	for _, ic := range g.ImageCVEs {
		ic.log(1)
	}
}

// DebugNode is a stripped down node to troubleshoot graph DB linking issues.
type DebugNode struct {
	ID          string
	Name        string
	ClusterID   string
	ClusterName string
}

func (n *DebugNode) log(offset int) {
	log.Infof(
		"%s* node {ID: %q, Name: %q, Cluster ID: %q, Cluster Name: %q}",
		getIndent(offset),
		n.ID,
		n.Name,
		n.ClusterID,
		n.ClusterName,
	)
}

// DebugNodeComponentEdge is a stripped down node to component edge to
// troubleshoot graph DB linking issues.
type DebugNodeComponentEdge struct {
	ID              string
	NodeID          string
	NodeComponentID string
}

func (nce *DebugNodeComponentEdge) log(offset int) {
	log.Infof(
		"%s* node component edge {ID: %q, Node ID: %q, Component ID: %q}",
		getIndent(offset),
		nce.ID,
		nce.NodeID,
		nce.NodeComponentID,
	)
}

// DebugNodeComponent is a stripped down node component to troubleshoot
// graph DB linking issues.
type DebugNodeComponent struct {
	ID string
}

func (nc *DebugNodeComponent) log(offset int) {
	log.Infof(
		"%s* node component {ID: %q}",
		getIndent(offset),
		nc.ID,
	)
}

// DebugNodeComponentCVEEdge is a stripped down node component to cve edge
// to troubleshoot graph DB linking issues.
type DebugNodeComponentCVEEdge struct {
	ID              string
	NodeComponentID string
	NodeCVEID       string
}

func (cce *DebugNodeComponentCVEEdge) log(offset int) {
	log.Infof(
		"%s* node component to CVE edge {ID: %q, Node Component ID: %q, CVE ID: %q}",
		getIndent(offset),
		cce.ID,
		cce.NodeComponentID,
		cce.NodeCVEID,
	)
}

// DebugNodeCVE is a stripped down node CVE to troubleshoot graph DB linking
// issues.
type DebugNodeCVE struct {
	ID string
}

func (nc *DebugNodeCVE) log(offset int) {
	log.Infof(
		"%s* node CVE {ID: %q}",
		getIndent(offset),
		nc.ID,
	)
}

// DebugNodeGraph is a container for object lists to troubleshoot graph DB
// linking issues.
type DebugNodeGraph struct {
	Nodes                 []DebugNode
	NodeComponentEdges    []DebugNodeComponentEdge
	NodeComponents        []DebugNodeComponent
	NodeComponentCVEEdges []DebugNodeComponentCVEEdge
	NodeCVEs              []DebugNodeCVE
}

// Log writes the collected graph data in the logs.
func (g *DebugNodeGraph) Log() {
	log.Info("Node graph:")
	log.Infof("- %d nodes", len(g.Nodes))
	for _, n := range g.Nodes {
		n.log(1)
	}
	log.Infof("- %d node to component edges", len(g.NodeComponentEdges))
	for _, nce := range g.NodeComponentEdges {
		nce.log(1)
	}
	log.Infof("- %d node components", len(g.NodeComponents))
	for _, nc := range g.NodeComponents {
		nc.log(1)
	}
	log.Infof("- %d node component to CVE edges", len(g.NodeComponentCVEEdges))
	for _, cce := range g.NodeComponentCVEEdges {
		cce.log(1)
	}
	log.Infof("- %d node CVEs", len(g.NodeCVEs))
	for _, nc := range g.NodeCVEs {
		nc.log(1)
	}
}

// GetImageGraph retrieves a stripped down dump of DB entries to troubleshoot
// graph DB linking issues.
func GetImageGraph(ctx context.Context, t *testing.T, db postgres.DB) DebugImageGraph {
	var graph DebugImageGraph
	graph.Deployments = listDeployments(ctx, t, db)
	graph.Containers = listContainers(ctx, t, db)
	graph.Images = listImages(ctx, t, db)
	graph.ImageComponentEdges = listImageToComponentEdges(ctx, t, db)
	graph.ImageComponents = listImageComponents(ctx, t, db)
	graph.ImageComponentCVEEdges = listImageComponentToCVEEdges(ctx, t, db)
	graph.ImageCVEEdges = listImageToCVEEdges(ctx, t, db)
	graph.ImageCVEs = listImageCVEs(ctx, t, db)
	return graph
}

func listDeployments(ctx context.Context, t *testing.T, db postgres.DB) []DebugDeployment {
	const selectStmt = "select id, name, namespace, clusterid, clustername from deployments"
	const fieldCount = 5
	populate := func(deployment *DebugDeployment, r [][]byte) {
		deployment.ID = uuid.FromBytesOrNil(r[0]).String()
		deployment.Name = string(r[1])
		deployment.Namespace = string(r[2])
		deployment.ClusterID = uuid.FromBytesOrNil(r[3]).String()
		deployment.ClusterName = string(r[4])
	}
	return populateListFromDB[DebugDeployment](ctx, t, db, selectStmt, fieldCount, populate)
}

func listContainers(ctx context.Context, t *testing.T, db postgres.DB) []DebugContainer {
	const selectStmt = "select deployments_id, idx, image_id from deployments_containers"
	const fieldCount = 3
	populate := func(container *DebugContainer, r [][]byte) {
		container.DeploymentID = uuid.FromBytesOrNil(r[0]).String()
		container.ImageID = string(r[2])
		identifier, err := strconv.ParseInt(string(r[1]), 10, 64)
		if err != nil {
			return
		}
		container.Idx = int(identifier)
	}
	return populateListFromDB[DebugContainer](ctx, t, db, selectStmt, fieldCount, populate)
}

func listImages(ctx context.Context, t *testing.T, db postgres.DB) []DebugImage {
	const selectStmt = "select id from images"
	const fieldCount = 1
	populate := func(image *DebugImage, r [][]byte) {
		image.ID = string(r[0])
	}
	return populateListFromDB[DebugImage](ctx, t, db, selectStmt, fieldCount, populate)
}

func listImageToComponentEdges(ctx context.Context, t *testing.T, db postgres.DB) []DebugImageComponentEdge {
	const selectStmt = "select id, imageid, imagecomponentid from image_component_edges"
	const fieldCount = 3
	populate := func(edge *DebugImageComponentEdge, r [][]byte) {
		edge.ID = string(r[0])
		edge.ImageID = string(r[1])
		edge.ImageComponentID = string(r[2])
	}
	return populateListFromDB[DebugImageComponentEdge](ctx, t, db, selectStmt, fieldCount, populate)
}

func listImageComponents(ctx context.Context, t *testing.T, db postgres.DB) []DebugImageComponent {
	const selectStmt = "select id from image_components"
	const fieldCount = 1
	populate := func(component *DebugImageComponent, r [][]byte) {
		component.ID = string(r[0])
	}
	return populateListFromDB[DebugImageComponent](ctx, t, db, selectStmt, fieldCount, populate)
}

func listImageComponentToCVEEdges(ctx context.Context, t *testing.T, db postgres.DB) []DebugImageComponentCVEEdge {
	const selectStmt = "select id, imagecomponentid, imagecveid from image_component_cve_edges"
	const fieldCount = 3
	populate := func(edge *DebugImageComponentCVEEdge, r [][]byte) {
		edge.ID = string(r[0])
		edge.ImageComponentID = string(r[1])
		edge.ImageCVEID = string(r[2])
	}
	return populateListFromDB[DebugImageComponentCVEEdge](ctx, t, db, selectStmt, fieldCount, populate)
}

func listImageToCVEEdges(ctx context.Context, t *testing.T, db postgres.DB) []DebugImageCVEEdge {
	const selectStmt = "select id, imageid, imagecveid, state from image_cve_edges"
	const fieldCount = 4
	populate := func(edge *DebugImageCVEEdge, r [][]byte) {
		edge.ID = string(r[0])
		edge.ImageID = string(r[1])
		edge.ImageCVEID = string(r[2])
		state, err := strconv.ParseInt(string(r[3]), 10, 64)
		if err != nil {
			return
		}
		edge.State = int32(state)
	}
	return populateListFromDB[DebugImageCVEEdge](ctx, t, db, selectStmt, fieldCount, populate)
}

func listImageCVEs(ctx context.Context, t *testing.T, db postgres.DB) []DebugImageCVE {
	const selectStmt = "select id from image_cves"
	const fieldCount = 1
	populate := func(vulnerability *DebugImageCVE, r [][]byte) {
		vulnerability.ID = string(r[0])
	}
	return populateListFromDB[DebugImageCVE](ctx, t, db, selectStmt, fieldCount, populate)
}

// GetNodeGraph retrieves a stripped down dump of DB entries to troubleshoot
// graph DB linking issues.
func GetNodeGraph(ctx context.Context, t *testing.T, db postgres.DB) DebugNodeGraph {
	var graph DebugNodeGraph
	graph.Nodes = listNodes(ctx, t, db)
	graph.NodeComponentEdges = listNodeToComponentEdges(ctx, t, db)
	graph.NodeComponents = listNodeComponents(ctx, t, db)
	graph.NodeComponentCVEEdges = listNodeComponentToCVEEdges(ctx, t, db)
	graph.NodeCVEs = listNodeCVEs(ctx, t, db)
	return graph
}

func listNodes(ctx context.Context, t *testing.T, db postgres.DB) []DebugNode {
	const selectStmt = "select id, name, cluster_id, cluster_name from nodes"
	const fieldCount = 1
	populate := func(node *DebugNode, r [][]byte) {
		node.ID = uuid.FromBytesOrNil(r[0]).String()
		node.Name = string(r[1])
		node.ClusterID = uuid.FromBytesOrNil(r[2]).String()
		node.ClusterName = string(r[3])
	}
	return populateListFromDB[DebugNode](ctx, t, db, selectStmt, fieldCount, populate)
}

func listNodeToComponentEdges(ctx context.Context, t *testing.T, db postgres.DB) []DebugNodeComponentEdge {
	const selectStmt = "select id, nodeid, nodecomponentid from node_component_edges"
	const fieldCount = 3
	populate := func(edge *DebugNodeComponentEdge, r [][]byte) {
		edge.ID = string(r[0])
		edge.NodeID = uuid.FromBytesOrNil(r[1]).String()
		edge.NodeComponentID = string(r[2])
	}
	return populateListFromDB[DebugNodeComponentEdge](ctx, t, db, selectStmt, fieldCount, populate)
}

func listNodeComponents(ctx context.Context, t *testing.T, db postgres.DB) []DebugNodeComponent {
	const selectStmt = "select id from node_components"
	const fieldCount = 1
	populate := func(component *DebugNodeComponent, r [][]byte) {
		component.ID = string(r[0])
	}
	return populateListFromDB[DebugNodeComponent](ctx, t, db, selectStmt, fieldCount, populate)
}

func listNodeComponentToCVEEdges(ctx context.Context, t *testing.T, db postgres.DB) []DebugNodeComponentCVEEdge {
	const selectStmt = "select id, nodecomponentid, nodecveid from node_components_cves_edges"
	const fieldCount = 3
	populate := func(edge *DebugNodeComponentCVEEdge, r [][]byte) {
		edge.ID = string(r[0])
		edge.NodeComponentID = string(r[1])
		edge.NodeCVEID = string(r[2])
	}
	return populateListFromDB[DebugNodeComponentCVEEdge](ctx, t, db, selectStmt, fieldCount, populate)
}

func listNodeCVEs(ctx context.Context, t *testing.T, db postgres.DB) []DebugNodeCVE {
	const selectStmt = "select id from node_cves"
	const fieldCount = 1
	populate := func(vulnerability *DebugNodeCVE, r [][]byte) {
		vulnerability.ID = string(r[0])
	}
	return populateListFromDB[DebugNodeCVE](ctx, t, db, selectStmt, fieldCount, populate)
}

func populateListFromDB[T any](ctx context.Context, _ *testing.T, db postgres.DB, selectStmt string, fieldCount int, populate func(obj *T, row [][]byte)) []T {
	results := make([]T, 0)
	rows, err := db.Query(sac.WithAllAccess(ctx), selectStmt)
	if err != nil {
		log.Info("Query \"", selectStmt, "\" failed with error ", err)
		return results
	}
	defer rows.Close()
	for rows.Next() {
		values := rows.RawValues()
		if len(values) < fieldCount {
			continue
		}
		var obj T
		populate(&obj, values)
		results = append(results, obj)
	}
	return results
}

func getIndent(offset int) string {
	var indent strings.Builder
	for i := 0; i < offset; i++ {
		indent.WriteString("  ")
	}
	return indent.String()
}
