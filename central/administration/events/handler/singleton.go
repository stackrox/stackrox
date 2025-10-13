package handler

import (
	"github.com/stackrox/rox/central/administration/events/datastore"
	"github.com/stackrox/rox/pkg/administration/events/stream"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	h Handler
)

// Singleton returns an instance of the administration events handler.
func Singleton() Handler {
	once.Do(func() {
		h = newHandler(datastore.Singleton(), stream.Singleton())
	})
	return h
}
