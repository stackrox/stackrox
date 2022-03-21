package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ImageComponentEdge --table=image_component_relation --search-category IMAGE_COMPONENT_EDGE --options-path central/imagecomponentedge/mappings  --referenced-tables=images,image_component --referenced-types=storage.Image,storage.ImageComponent --skip-mutators
