package inmem

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/db/boltdb"
	"bitbucket.org/stack-rox/apollo/central/search/blevesearch"
)

func createBoltDB() (db.Storage, error) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, fmt.Errorf("Failed to get temporary directory: %v", err.Error())
	}
	indexer, err := blevesearch.NewIndexer()
	if err != nil {
		return nil, err
	}
	db, err := boltdb.New(filepath.Join(tmpDir, "prevent.db"), indexer)
	if err != nil {
		return nil, err
	}
	return db, err
}
