package gocache

import "log"

func logf(format string, args ...interface{}) {
	log.Printf("cimemo-gocache: "+format, args...)
}
