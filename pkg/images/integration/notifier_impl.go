package integration

import (
	"sync"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
)

type notifierImpl struct {
	lock sync.Mutex

	onUpdates []func(*storage.ImageIntegration) error
	onRemoves []func(id string) error
}

// NotifyUpdated notifies the receivers of an updated image integration.
func (c *notifierImpl) NotifyUpdated(integration *storage.ImageIntegration) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	errList := errorhelpers.NewErrorList("notifying of update")
	for _, onUpdate := range c.onUpdates {
		if err := onUpdate(integration); err != nil {
			errList.AddError(err)
		}
	}
	return errList.ToError()
}

// NotifyRemoved notifies the receivers of an removed image integration.
func (c *notifierImpl) NotifyRemoved(id string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	errList := errorhelpers.NewErrorList("notifying of removal")
	for _, onRemove := range c.onRemoves {
		if err := onRemove(id); err != nil {
			errList.AddError(err)
		}
	}
	return errList.ToError()
}

// addOnUpdate adds a receiver for updates.
func (c *notifierImpl) addOnUpdate(onUpdate func(*storage.ImageIntegration) error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.onUpdates = append(c.onUpdates, onUpdate)
}

// addOnRemove adds a receiver for removals.
func (c *notifierImpl) addOnRemove(onRemove func(string) error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.onRemoves = append(c.onRemoves, onRemove)
}
