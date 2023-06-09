package globaldb

// Close closes all global databases. Should only be used at central shutdown time.
func Close() {
	if rocksDB != nil {
		rocksDB.Close()
	}
	if postgresDB != nil {
		postgresDB.Close()
	}
}
