package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ActiveComponent --search-category ACTIVE_COMPONENT --references storage.Deployment,storage.ImageComponent --postgres-migration-seq 5 --migrate-from "dackbox:active_components"
