package postgres

//go:generate pg-table-bindings-wrapper --type=storage.IntegrationHealth --permission-checker permissionCheckerSingleton() --postgres-migration-seq 25 --migrate-from "rocksdb:integrationhealth"
