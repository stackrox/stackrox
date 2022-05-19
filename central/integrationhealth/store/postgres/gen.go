ackage postgres

//go:generate pg-table-bindings-wrapper --type=storage.IntegrationHealth --permission-checker permissionCheckerSingleton()
