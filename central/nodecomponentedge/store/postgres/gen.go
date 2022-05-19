package postgres

//go:generate pg-table-bindings-wrapper --type=storage.NodeComponentEdge --search-category NODE_COMPONENT_EDGE --references=storage.Node,storage.ImageComponent --join-table true
