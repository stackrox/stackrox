package globaldb

// Close closes all global databases. Should only be used at central shutdown time.
func Close() {
	if postgresDB != nil {
		postgresDB.Close()
	}
}
