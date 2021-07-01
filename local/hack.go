package main

import (
	"path/filepath"

	"github.com/stackrox/rox/central/cve/datastore"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/option"
)

func main() {
	option.CentralOptions.DBPathBase = "local/database-restore/full"

	blevePath := filepath.Join(option.CentralOptions.DBPathBase, "bleve")
	globalindex.DefaultBlevePath = filepath.Join(blevePath, "default")
	globalindex.DefaultTmpBlevePath = filepath.Join(blevePath, "tmp")
	globalindex.SeparateIndexPath = filepath.Join(blevePath, "separate")

	// Can start accessing _most_ singletons. Some singletons that access certificates will fail
	datastore.Singleton()
}
