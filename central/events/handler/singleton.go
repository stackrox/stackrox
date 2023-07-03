package handler

import (
	"github.com/stackrox/rox/central/events/datastore"
	"github.com/stackrox/rox/pkg/events"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	h events.Handler
)

// Singleton returns an instance of the event handler.
func Singleton() events.Handler {
	once.Do(func() {
		h = newHandler(datastore.Singleton())
	})
	return h
}
