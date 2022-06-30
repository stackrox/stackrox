package postgres

import "github.com/stackrox/rox/pkg/postgres/schema"

var (
	nodeTree = map[string]struct{}{
		schema.NodeCvesTableName:                {},
		schema.NodeComponentsCvesEdgesTableName: {},
		schema.NodeComponentsTableName:          {},
		schema.NodeComponentEdgesTableName:      {},
		schema.NodesTableName:                   {},
		schema.ClustersTableName:                {},
	}

	imageTree = map[string]struct{}{
		schema.ImageCvesTableName:              {},
		schema.ImageComponentCveEdgesTableName: {},
		schema.ImageComponentsTableName:        {},
		schema.ImageComponentEdgesTableName:    {},
		schema.ImageCveEdgesTableName:          {},
		schema.ImagesTableName:                 {},
		schema.DeploymentsTableName:            {},
		schema.NamespacesTableName:             {},
		schema.ClustersTableName:               {},
	}

	searchNamespace = map[string]map[string]struct{}{
		schema.ImageCvesTableName:              imageTree,
		schema.ImageComponentCveEdgesTableName: imageTree,
		schema.ImageComponentsTableName:        imageTree,
		schema.ImageComponentEdgesTableName:    imageTree,
		schema.ImagesTableName:                 imageTree,
		schema.DeploymentsTableName:            imageTree,
		schema.NamespacesTableName:             imageTree,

		schema.NodeCvesTableName:                nodeTree,
		schema.NodeComponentsCvesEdgesTableName: nodeTree,
		schema.NodeComponentsTableName:          nodeTree,
		schema.NodeComponentEdgesTableName:      nodeTree,
		schema.NodesTableName:                   nodeTree,

		// for testing

		schema.TestParent4TableName: {
			schema.TestGrandparentsTableName: {},
			schema.TestChild1P4TableName:     {},
		},
		schema.TestChild1P4TableName: {
			schema.TestChild1P4TableName: {},
		},
	}
)
