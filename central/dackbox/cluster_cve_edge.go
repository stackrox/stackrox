package dackbox

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/dackbox/keys/transformation"
)

var (
	// ClusterCVEEdgeTransformations holds the transformations to go from a cluster:cve edge id to the ids of the given category.
	ClusterCVEEdgeTransformations = map[v1.SearchCategory]transformation.OneToMany{
		// Edge (parse first key in pair) Cluster
		v1.SearchCategory_CLUSTERS: transformation.Split([]byte(":")).
			ThenMapEachToOne(transformation.Decode()).
			Then(transformation.AtIndex(0)),

		// We don't want to map cluster vulns to objects within the cluster.
		v1.SearchCategory_NAMESPACES:           ReturnNothing,
		v1.SearchCategory_DEPLOYMENTS:          ReturnNothing,
		v1.SearchCategory_ACTIVE_COMPONENT:     ReturnNothing,
		v1.SearchCategory_IMAGES:               ReturnNothing,
		v1.SearchCategory_IMAGE_VULN_EDGE:      ReturnNothing,
		v1.SearchCategory_IMAGE_COMPONENT_EDGE: ReturnNothing,
		v1.SearchCategory_IMAGE_COMPONENTS:     ReturnNothing,
		v1.SearchCategory_NODES:                ReturnNothing,
		v1.SearchCategory_NODE_VULN_EDGE:       ReturnNothing,
		v1.SearchCategory_NODE_COMPONENT_EDGE:  ReturnNothing,
		v1.SearchCategory_COMPONENT_VULN_EDGE:  ReturnNothing,

		// Edge (parse second key in pair) CVE
		v1.SearchCategory_VULNERABILITIES: transformation.Split([]byte(":")).
			ThenMapEachToOne(transformation.Decode()).
			Then(transformation.AtIndex(1)),

		// Edge
		v1.SearchCategory_CLUSTER_VULN_EDGE: DoNothing,
	}
)
