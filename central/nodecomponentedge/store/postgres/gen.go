package postgres

//go:generate pg-table-bindings-wrapper --type=storage.NodeComponentEdge --search-category NODE_COMPONENT_EDGE --references=storage.Node,node_components:storage.NodeComponent --join-table true
