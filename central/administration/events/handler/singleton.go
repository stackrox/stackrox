package handler

import (
	"github.com/stackrox/rox/central/administration/events/datastore"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	h Handler
)

// Singleton returns an instance of the administration events handler.
func Singleton() Handler {
	if !features.AdministrationEvents.Enabled() {
		return nil
	}
	once.Do(func() {
		h = newHandler(datastore.Singleton(), events.Singleton())
	})
	return h
}
