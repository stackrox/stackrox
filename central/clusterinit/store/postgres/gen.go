package postgres

//go:generate pg-table-bindings-wrapper --type=storage.InitBundleMeta --table=clusterinitbundles --permission-checker permissionCheckerSingleton()
