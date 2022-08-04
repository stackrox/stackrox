package rocksdb

//go:generate rocksdb-bindings-wrapper --type=ReportConfiguration --bucket=report_configs --cache --key-func GetId() --migration-seq 43 --migrate-to report_configurations
