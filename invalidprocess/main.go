package main

import (
	"context"
	"os"
	"strconv"
	"unicode/utf8"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/processindicator/store/rocksdb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/stringutils"
)

var log = logging.LoggerForModule()

func main() {
	shouldDelete := os.Getenv("DELETE_PROCESSES") == "true"
	maxPrintString := stringutils.OrDefault(os.Getenv("MAX_PRINT"), "5")
	numToPrint, err := strconv.Atoi(maxPrintString)
	if err != nil {
		log.Errorf("Could not parse max print string. defaulting to 5")
		numToPrint = 5
	}

	ctx := sac.WithAllAccess(context.Background())

	store := rocksdb.New(globaldb.GetRocksDB())
	defer globaldb.GetRocksDB().Close()

	var idsToDelete []string

	err = store.Walk(ctx, func(obj *storage.ProcessIndicator) error {
		var bad bool
		if !utf8.ValidString(obj.GetSignal().GetExecFilePath()) {
			bad = true
			if numToPrint > 0 {
				log.Errorf("Found non utf-8 in exec file path for %s", obj)
			}
		}
		if !utf8.ValidString(obj.GetSignal().GetName()) {
			bad = true
			if numToPrint > 0 {
				log.Errorf("Found non utf-8 in name for %s", obj)
			}
		}
		if !utf8.ValidString(obj.GetSignal().GetArgs()) {
			bad = true
			if numToPrint > 0 {
				log.Errorf("Found non utf-8 in args for %s", obj)
			}
		}
		for _, lineage := range obj.GetSignal().GetLineageInfo() {
			if !utf8.ValidString(lineage.GetParentExecFilePath()) {
				bad = true
				if numToPrint > 0 {
					log.Errorf("Found non utf-8 in lineage for %s", obj)
				}
			}
		}
		if bad {
			numToPrint--
			idsToDelete = append(idsToDelete, obj.GetId())
		}
		return nil
	})
	if err != nil {
		log.Fatalf("error walking processes: %v", err)
	}
	log.Infof("found %d processes that are not properly UTF8 encoded", len(idsToDelete))
	if !shouldDelete {
		return
	}

	if err := store.DeleteMany(ctx, idsToDelete); err != nil {
		log.Fatalf("error deleting processes: %v", err)
	}
	log.Infof("Succcessfully deleted %d processes", len(idsToDelete))
}
