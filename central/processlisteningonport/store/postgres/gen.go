package postgres

//go:generate pg-table-bindings-wrapper --type=storage.ProcessListeningOnPortStorage --references storage.ProcessIndicator,storage.Deployment,storage.Pod --table=listening_endpoints --search-category PROCESS_LISTENING_ON_PORT
