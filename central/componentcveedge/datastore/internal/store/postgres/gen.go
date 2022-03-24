package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ComponentCVEEdge --table=component_image_cve_relation --search-category COMPONENT_VULN_EDGE --options-path central/componentcveedge/mappings  --references=image_components:storage.ImageComponent(image_component_id:id),image_cves:storage.CVE(cve_id:id;cve_operating_system:operating_system) --join-table
