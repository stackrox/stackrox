package rocksdb

//go:generate rocksdb-bindings-wrapper --type=SignatureIntegration --bucket=signature_integrations --cache --uniq-key-func GetName() --migrate-seq 51 --migrate-to signature_integrations
