package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ImageComponentEdge --table=image_component_relations --search-category IMAGE_COMPONENT_EDGE --options-path central/imagecomponentedge/mappings  --references=images:storage.Image,image_components:storage.ImageComponent --join-table true
