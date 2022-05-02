package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ImageComponentEdge --table=image_component_relations --search-category IMAGE_COMPONENT_EDGE --references=images:storage.Image,image_components:storage.ImageComponent --join-table true
