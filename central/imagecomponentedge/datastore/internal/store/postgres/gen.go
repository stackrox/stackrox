package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ImageComponentEdge --search-category IMAGE_COMPONENT_EDGE --references=storage.Image,storage.ImageComponent --join-table true
