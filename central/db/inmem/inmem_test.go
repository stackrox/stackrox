package inmem

import (
	"fmt"
	"io/ioutil"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/db/boltdb"
)

func createBoltDB() (db.Storage, error) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, fmt.Errorf("Failed to get temporary directory: %v", err.Error())
	}
	db, err := boltdb.New(tmpDir)
	if err != nil {
		return nil, err
	}
	return db, err
}
