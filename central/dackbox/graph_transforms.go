package dackbox

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/dackbox/keys/transformation"
)

/*
The DackBox graph is currently designed with the following directed hierarchy:

            -Cluster ID----------------
          /     |                       \
         /      V                        \
        /   Namespace ID                  |
       /        |                         |
      /         V                         |
     |      Deployment ID-----            |
     |          |             |           |
     |          V             V           |
  Node IDs   Image IDs   ActiveComponent  |
     |          |             /           |
      \         V            /            |
        --> Component IDs <-              |
                |                        /
                V                       /
             CVE IDs <-----------------

So to get from a cluster to it's CVEs, there are two paths (which lead to the two different kinds of CVEs).
One from the deployment pointing toward it:
Cluster (forwards) Namespaces (forwards) Deployments (forwards) Images (forwards) Components (forwards) CVEs
And one direct from the cluster to the CVEs:
Cluster (fowards) CVEs

Note:
- The CVEs pointed to by components, and those pointed to by clusters, are likely disjoint sets.
- Direct links between Image and CVEs bypass the graph (for performance reasons).
*/

var (
	// DoNothing is a OneToMany that returns the input ID.
	DoNothing = func(_ context.Context, key []byte) [][]byte { return [][]byte{key} }

	// ReturnNothing is a OneToMany that returns no IDs, therefore blocking the mapping.
	ReturnNothing = func(_ context.Context, key []byte) [][]byte { return nil }

	// GraphTransformations holds how to scope a secondary category under a primary category.
	// For instance, if you want to search CVEs within the scope of an image, you would use the function stored
	// under GraphTransformations[v1.SearchCategory_IMAGES][v1.SearchCategory_VULNERABILITIES] to pull the vulns
	// that exist in the image.
	GraphTransformations = map[v1.SearchCategory]map[v1.SearchCategory]transformation.OneToMany{
		v1.SearchCategory_CLUSTERS:             ClusterTransformations,
		v1.SearchCategory_NAMESPACES:           NamespaceTransformations,
		v1.SearchCategory_NODES:                NodeTransformations,
		v1.SearchCategory_NODE_VULN_EDGE:       NodeCVEEdgeTransformations,
		v1.SearchCategory_NODE_COMPONENT_EDGE:  NodeComponentEdgeTransformations,
		v1.SearchCategory_DEPLOYMENTS:          DeploymentTransformations,
		v1.SearchCategory_ACTIVE_COMPONENT:     ActiveComponentTransformations,
		v1.SearchCategory_IMAGES:               ImageTransformations,
		v1.SearchCategory_IMAGE_VULN_EDGE:      ImageCVEEdgeTransformations,
		v1.SearchCategory_IMAGE_COMPONENT_EDGE: ImageComponentEdgeTransformations,
		v1.SearchCategory_IMAGE_COMPONENTS:     ComponentTransformations,
		v1.SearchCategory_COMPONENT_VULN_EDGE:  ComponentCVEEdgeTransformations,
		v1.SearchCategory_VULNERABILITIES:      CVETransformations,
		v1.SearchCategory_CLUSTER_VULN_EDGE:    ClusterCVEEdgeTransformations,
	}
)

// ToCategory returns a transformation provider that transforms to the input category from some other type.
func ToCategory(cat v1.SearchCategory) TransformationProvider {
	return fromTransformationProviderImpl{
		transformations: GraphTransformations,
		secondary:       cat,
	}
}

// TransformationProvider provides a transformation.OneToMany for a given input category.
type TransformationProvider interface {
	Get(v1.SearchCategory) transformation.OneToMany
}

type fromTransformationProviderImpl struct {
	transformations map[v1.SearchCategory]map[v1.SearchCategory]transformation.OneToMany
	secondary       v1.SearchCategory
}

func (ftp fromTransformationProviderImpl) Get(sc v1.SearchCategory) transformation.OneToMany {
	return ftp.transformations[sc][ftp.secondary]
}
