package postgres

//go:generate pgsearchbindings-wrapper --write-options=false --options-path "central/secret/mappings" --type Secret --singular Secret --search-category SECRETS
