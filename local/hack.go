package main

import (
	"path/filepath"

	"github.com/stackrox/stackrox/central/cve/datastore"
	"github.com/stackrox/stackrox/central/globalindex"
	"github.com/stackrox/stackrox/central/option"
	"github.com/stackrox/stackrox/pkg/features"
)

func main() {
	if features.PostgresDatastore.Enabled() {
		return
	}

	option.CentralOptions.DBPathBase = "local/database-restore/full"

	blevePath := filepath.Join(option.CentralOptions.DBPathBase, "bleve")
	globalindex.DefaultBlevePath = filepath.Join(blevePath, "default")
	globalindex.DefaultTmpBlevePath = filepath.Join(blevePath, "tmp")
	globalindex.SeparateIndexPath = filepath.Join(blevePath, "separate")

	// Can start accessing _most_ singletons. Some singletons that access certificates will fail
	datastore.Singleton()
}
