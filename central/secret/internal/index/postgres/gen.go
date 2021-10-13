package postgres

//go:generate pgsearchbindings-wrapper --table secrets --write-options=false --options-path "central/secret/mappings" --type Secret --singular Secret --search-category SECRETS
