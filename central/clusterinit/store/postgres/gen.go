package postgres

//go:generate pg-table-bindings-wrapper --type=storage.InitBundleMeta --cached-store --table=cluster_init_bundles --permission-checker sac.NewAllGlobalResourceAllowedPermissionChecker(resources.Administration,resources.Integration)
