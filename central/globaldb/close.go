package globaldb

// Close closes all global databases. Should only be used at central shutdown time.
func Close() {
	if globalDB != nil {
		if err := globalDB.Close(); err != nil {
			log.Errorf("Unable to close bolt db: %v", err)
		}
	}
	if rocksDB != nil {
		rocksDB.Close()
	}
	if postgresDB != nil {
		postgresDB.Close()
	}
}
