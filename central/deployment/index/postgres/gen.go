package postgres

//go:generate pgsearchbindings-wrapper --object Deployment --singular deployment --search-category DEPLOYMENTS
//go:generate mockgen-wrapper Indexer
