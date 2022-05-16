package postgres

//go:generate pg-table-bindings-wrapper --type=storage.NodeComponentEdge --table=nodes_to_components --search-category NODE_COMPONENT_EDGE --references=nodes:storage.Node,node_components:storage.ImageComponent --join-table true
