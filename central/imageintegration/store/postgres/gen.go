package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ImageIntegration --search-category IMAGE_INTEGRATIONS --references storage.ImageIntegration --get-all-func
