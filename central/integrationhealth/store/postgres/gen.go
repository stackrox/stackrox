package postgres

//go:generate pg-table-bindings-wrapper --type=storage.IntegrationHealth --table=integration_healths --permission-checker permissionCheckerSingleton()
