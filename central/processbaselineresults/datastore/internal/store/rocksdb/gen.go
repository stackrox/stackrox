package rocksdb

//go:generate rocksdb-bindings-wrapper --type=ProcessBaselineResults --bucket=processWhitelistResults --key-func=GetDeploymentId() --migration-seq 40 --migrate-to process_baseline_results
