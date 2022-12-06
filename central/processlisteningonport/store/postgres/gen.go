package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ProcessListeningOnPortStorage --references storage.ProcessIndicator --table=process_listening_on_ports --search-category PROCESS_LISTENING_ON_PORT --schema-only
