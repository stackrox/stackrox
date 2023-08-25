package handler

import (
	"github.com/stackrox/rox/central/notifications/datastore"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/notifications"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	h Handler
)

// Singleton returns an instance of the notification handler.
func Singleton() Handler {
	if !features.CentralNotifications.Enabled() {
		return nil
	}
	once.Do(func() {
		h = newHandler(datastore.Singleton(), notifications.Singleton())
	})
	return h
}
