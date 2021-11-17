package postgres

//go:generate pgsearchbindings-wrapper --table alert --write-options=false --options-path "central/alert/mappings" --type ListAlert --singular ListAlert --search-category ALERTS
