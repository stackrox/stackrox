package postgres

//go:generate pgsearchbindings-wrapper --type Deployment --table deployment --search-category DEPLOYMENTS --options-path "pkg/search/options/deployments"
// //go:generate mockgen-wrapper Indexer
