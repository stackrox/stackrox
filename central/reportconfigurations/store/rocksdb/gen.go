package rocksdb

//go:generate rocksdb-bindings-wrapper --type=ReportConfiguration --bucket=report_configs --cache --key-func GetId()
