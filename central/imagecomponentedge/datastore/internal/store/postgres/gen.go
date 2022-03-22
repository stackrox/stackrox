package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ImageComponentEdge --table=image_component_relation --search-category IMAGE_COMPONENT_EDGE --options-path central/imagecomponentedge/mappings  --references=images:storage.Image,image_component:storage.ImageComponent --join-table true
