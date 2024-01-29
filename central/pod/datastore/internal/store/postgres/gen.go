package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Pod --cached-store --search-category PODS --references storage.Deployment
