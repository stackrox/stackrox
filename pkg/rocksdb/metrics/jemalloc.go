package metrics

/*
// This cgo directive is what actually causes jemalloc to be linked in to the
// final Go executable
#cgo pkg-config: jemalloc
#include <jemalloc/jemalloc.h>
void _refresh_jemalloc_stats() {
	// You just need to pass something not-null into the "epoch" mallctl.
	size_t random_something = 1;
	mallctl("epoch", NULL, NULL, &random_something, sizeof(random_something));
}
int _get_jemalloc_active() {
	size_t stat, stat_size;
	stat = 0;
	stat_size = sizeof(stat);
	mallctl("stats.active", &stat, &stat_size, NULL, 0);
	return (int)stat;
}
*/
import "C"

import (
	"time"

	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

func init() {
	log.Info("loading jemalloc")

	go func() {
		t := time.NewTicker(5 * time.Second)
		for range t.C {
			C._refresh_jemalloc_stats()
			log.Infof("JEMALLOC: %0.2f MB", float64(C._get_jemalloc_active())/1024/1024)
			jemallocAllocations.Set(float64(C._get_jemalloc_active()))
		}
	}()
}
