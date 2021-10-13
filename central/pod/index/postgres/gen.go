package postgres

//go:generate pgsearchbindings-wrapper --table pods --write-options=false --options-path "central/pod/mappings" --singular Pod --type Pod --search-category PODS
