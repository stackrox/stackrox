package testutils

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac"
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
	ID           string
	ImageID      string
}

func (c *DebugContainer) log(offset int) {
	log.Infof(
		"%s* container {Deployment ID %q, Index %q, Image ID %q}",
		getIndent(offset),
		c.DeploymentID,
		c.ID,
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

func listDeployments(ctx context.Context, _ *testing.T, db postgres.DB) []DebugDeployment {
	results := make([]DebugDeployment, 0)
	const selectStmt = "select id, name, namespace, clusterid, clustername from deployments"
	rows, err := db.Query(sac.WithAllAccess(ctx), selectStmt)
	if err != nil {
		return results
	}
	defer rows.Close()
	results = make([]DebugDeployment, 0, len(rows.RawValues()))
	for _, r := range rows.RawValues() {
		if len(r) < 5 {
			continue
		}
		var deployment DebugDeployment
		deployment.ID = string(r[0])
		deployment.Name = string(r[1])
		deployment.Namespace = string(r[2])
		deployment.ClusterID = string(r[3])
		deployment.ClusterName = string(r[4])
		results = append(results, deployment)
	}
	return results
}

func listContainers(ctx context.Context, _ *testing.T, db postgres.DB) []DebugContainer {
	results := make([]DebugContainer, 0)
	const selectStmt = "select deployment_id, idx, image_id from deployments_containers"
	rows, err := db.Query(sac.WithAllAccess(ctx), selectStmt)
	if err != nil {
		return results
	}
	defer rows.Close()
	results = make([]DebugContainer, 0, len(rows.RawValues()))
	for _, r := range rows.RawValues() {
		if len(r) < 3 {
			continue
		}
		var container DebugContainer
		container.DeploymentID = string(r[0])
		container.ID = string(r[1])
		container.ImageID = string(r[2])
		results = append(results, container)
	}
	return results
}

func listImages(ctx context.Context, _ *testing.T, db postgres.DB) []DebugImage {
	results := make([]DebugImage, 0)
	const selectStmt = "select id from images"
	rows, err := db.Query(sac.WithAllAccess(ctx), selectStmt)
	if err != nil {
		return results
	}
	results = make([]DebugImage, 0, len(rows.RawValues()))
	for _, r := range rows.RawValues() {
		if len(r) < 1 {
			continue
		}
		var image DebugImage
		image.ID = string(r[0])
		results = append(results, image)
	}
	return results
}

func listImageToComponentEdges(ctx context.Context, _ *testing.T, db postgres.DB) []DebugImageComponentEdge {
	results := make([]DebugImageComponentEdge, 0)
	const selectStmt = "select id, imageid, imagecomponentid from image_component_edges"
	rows, err := db.Query(sac.WithAllAccess(ctx), selectStmt)
	if err != nil {
		return results
	}
	results = make([]DebugImageComponentEdge, 0, len(rows.RawValues()))
	for _, r := range rows.RawValues() {
		if len(r) < 3 {
			continue
		}
		var edge DebugImageComponentEdge
		edge.ID = string(r[0])
		edge.ImageID = string(r[1])
		edge.ImageComponentID = string(r[2])
		results = append(results, edge)
	}
	return results
}

func listImageComponents(ctx context.Context, _ *testing.T, db postgres.DB) []DebugImageComponent {
	results := make([]DebugImageComponent, 0)
	const selectStmt = "select id from image_components"
	rows, err := db.Query(sac.WithAllAccess(ctx), selectStmt)
	if err != nil {
		return results
	}
	results = make([]DebugImageComponent, 0, len(rows.RawValues()))
	for _, r := range rows.RawValues() {
		if len(r) < 1 {
			continue
		}
		var component DebugImageComponent
		component.ID = string(r[0])
		results = append(results, component)
	}
	return results
}

func listImageComponentToCVEEdges(ctx context.Context, _ *testing.T, db postgres.DB) []DebugImageComponentCVEEdge {
	results := make([]DebugImageComponentCVEEdge, 0)
	const selectStmt = "select id, imagecomponentid, imagecveid from image_component_cve_edges"
	rows, err := db.Query(sac.WithAllAccess(ctx), selectStmt)
	if err != nil {
		return results
	}
	results = make([]DebugImageComponentCVEEdge, 0, len(rows.RawValues()))
	for _, r := range rows.RawValues() {
		if len(r) < 3 {
			continue
		}
		var edge DebugImageComponentCVEEdge
		edge.ID = string(r[0])
		edge.ImageComponentID = string(r[1])
		edge.ImageCVEID = string(r[2])
		results = append(results, edge)
	}
	return results
}

func listImageToCVEEdges(ctx context.Context, _ *testing.T, db postgres.DB) []DebugImageCVEEdge {
	results := make([]DebugImageCVEEdge, 0)
	const selectStmt = "select id, imageid, imagecveid, state from image_cve_edges"
	rows, err := db.Query(sac.WithAllAccess(ctx), selectStmt)
	if err != nil {
		return results
	}
	results = make([]DebugImageCVEEdge, 0, len(rows.RawValues()))
	for _, r := range rows.RawValues() {
		if len(r) < 4 {
			continue
		}
		var edge DebugImageCVEEdge
		edge.ID = string(r[0])
		edge.ImageID = string(r[1])
		edge.ImageCVEID = string(r[2])
		state, err := strconv.ParseInt(string(r[3]), 10, 64)
		if err != nil {
			continue
		}
		edge.State = int32(state)
		results = append(results, edge)
	}
	return results
}

func listImageCVEs(ctx context.Context, _ *testing.T, db postgres.DB) []DebugImageCVE {
	results := make([]DebugImageCVE, 0)
	const selectStmt = "select id from image_cves"
	rows, err := db.Query(sac.WithAllAccess(ctx), selectStmt)
	if err != nil {
		return results
	}
	results = make([]DebugImageCVE, 0, len(rows.RawValues()))
	for _, r := range rows.RawValues() {
		if len(r) < 1 {
			continue
		}
		var vulnerability DebugImageCVE
		vulnerability.ID = string(r[0])
		results = append(results, vulnerability)
	}
	return results
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

func listNodes(ctx context.Context, _ *testing.T, db postgres.DB) []DebugNode {
	results := make([]DebugNode, 0)
	const selectStmt = "select id from nodes"
	rows, err := db.Query(sac.WithAllAccess(ctx), selectStmt)
	if err != nil {
		return results
	}
	results = make([]DebugNode, 0, len(rows.RawValues()))
	for _, r := range rows.RawValues() {
		if len(r) < 1 {
			continue
		}
		var node DebugNode
		node.ID = string(r[0])
		results = append(results, node)
	}
	return results
}

func listNodeToComponentEdges(ctx context.Context, _ *testing.T, db postgres.DB) []DebugNodeComponentEdge {
	results := make([]DebugNodeComponentEdge, 0)
	const selectStmt = "select id, nodeid, nodecomponentid from node_component_edges"
	rows, err := db.Query(sac.WithAllAccess(ctx), selectStmt)
	if err != nil {
		return results
	}
	results = make([]DebugNodeComponentEdge, 0, len(rows.RawValues()))
	for _, r := range rows.RawValues() {
		if len(r) < 3 {
			continue
		}
		var edge DebugNodeComponentEdge
		edge.ID = string(r[0])
		edge.NodeID = string(r[1])
		edge.NodeComponentID = string(r[2])
		results = append(results, edge)
	}
	return results
}

func listNodeComponents(ctx context.Context, _ *testing.T, db postgres.DB) []DebugNodeComponent {
	results := make([]DebugNodeComponent, 0)
	const selectStmt = "select id from node_components"
	rows, err := db.Query(sac.WithAllAccess(ctx), selectStmt)
	if err != nil {
		return results
	}
	results = make([]DebugNodeComponent, 0, len(rows.RawValues()))
	for _, r := range rows.RawValues() {
		if len(r) < 1 {
			continue
		}
		var component DebugNodeComponent
		component.ID = string(r[0])
		results = append(results, component)
	}
	return results
}

func listNodeComponentToCVEEdges(ctx context.Context, _ *testing.T, db postgres.DB) []DebugNodeComponentCVEEdge {
	results := make([]DebugNodeComponentCVEEdge, 0)
	const selectStmt = "select id, nodecomponentid, nodecveid from node_components_cves_edges"
	rows, err := db.Query(sac.WithAllAccess(ctx), selectStmt)
	if err != nil {
		return results
	}
	results = make([]DebugNodeComponentCVEEdge, 0, len(rows.RawValues()))
	for _, r := range rows.RawValues() {
		if len(r) < 3 {
			continue
		}
		var edge DebugNodeComponentCVEEdge
		edge.ID = string(r[0])
		edge.NodeComponentID = string(r[1])
		edge.NodeCVEID = string(r[2])
		results = append(results, edge)
	}
	return results
}

func listNodeCVEs(ctx context.Context, _ *testing.T, db postgres.DB) []DebugNodeCVE {
	results := make([]DebugNodeCVE, 0)
	const selectStmt = "select id from node_cves"
	rows, err := db.Query(sac.WithAllAccess(ctx), selectStmt)
	if err != nil {
		return results
	}
	results = make([]DebugNodeCVE, 0, len(rows.RawValues()))
	for _, r := range rows.RawValues() {
		if len(r) < 1 {
			continue
		}
		var vulnerability DebugNodeCVE
		vulnerability.ID = string(r[0])
		results = append(results, vulnerability)
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
