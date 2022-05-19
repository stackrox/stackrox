package postgres

//go:generate pg-table-bindings-wrapper --type=storage.InitBundleMeta --permission-checker permissionCheckerSingleton()
