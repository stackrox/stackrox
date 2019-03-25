package globaldb

// Close closes all global databases. Should only be used at central shutdown time.
func Close() {
	boltDB := GetGlobalDB()
	if err := boltDB.Close(); err != nil {
		log.Errorf("Unable to close bolt db: %v", err)
	}
	badgerDB := GetGlobalBadgerDB()
	if err := badgerDB.Close(); err != nil {
		log.Errorf("Unable to close badger db: %v", err)
	}
}
