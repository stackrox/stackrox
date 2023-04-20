package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"unicode/utf8"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/processindicator/store/rocksdb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
)

var log = logging.LoggerForModule()

func main() {
	shouldDelete := os.Getenv("DELETE_PROCESSES") == "true"

	ctx := sac.WithAllAccess(context.Background())

	store := rocksdb.New(globaldb.GetRocksDB())
	defer globaldb.GetRocksDB().Close()

	var idsToDelete []string
	err := store.Walk(ctx, func(obj *storage.ProcessIndicator) error {
		var bad bool
		if !utf8.ValidString(obj.GetSignal().GetExecFilePath()) {
			bad = true
			log.Errorf("Found non utf-8 in exec file path for deployment=%s container=%s data=%s", obj.GetDeploymentId(), obj.GetContainerName(), obj.GetSignal().GetExecFilePath())
		}
		if !utf8.ValidString(obj.GetSignal().GetName()) {
			bad = true
			log.Errorf("Found non utf-8 in name for deployment=%s container=%s data=%s", obj.GetDeploymentId(), obj.GetContainerName(), obj.GetSignal().GetName())
		}
		if !utf8.ValidString(obj.GetSignal().GetArgs()) {
			bad = true
			log.Errorf("Found non utf-8 in args for deployment=%s container=%s data=%s", obj.GetDeploymentId(), obj.GetContainerName(), obj.GetSignal().GetArgs())
		}
		for _, lineage := range obj.GetSignal().GetLineageInfo() {
			if !utf8.ValidString(lineage.GetParentExecFilePath()) {
				bad = true
				log.Errorf("Found non utf-8 in lineage for deployment=%s container=%s data=%s", obj.GetDeploymentId(), obj.GetContainerName(), lineage.GetParentExecFilePath())
			}
		}
		if bad {
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

	signalsC := make(chan os.Signal, 1)
	signal.Notify(signalsC, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	sig := <-signalsC
	log.Infof("Caught %s signal", sig)
}
