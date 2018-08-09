package integration

import (
	"sync"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errorhelpers"
)

type notifierImpl struct {
	lock sync.Mutex

	onUpdates []func(*v1.ImageIntegration) error
	onRemoves []func(id string) error
}

// NotifyUpdated notifies the receivers of an updated image integration.
func (c *notifierImpl) NotifyUpdated(integration *v1.ImageIntegration) error {
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

// AddOnUpdate adds a receiver for updates.
func (c *notifierImpl) AddOnUpdate(onUpdate func(*v1.ImageIntegration) error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.onUpdates = append(c.onUpdates, onUpdate)
}

// AddOnRemove adds a receiver for removals.
func (c *notifierImpl) AddOnRemove(onRemove func(string) error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.onRemoves = append(c.onRemoves, onRemove)
}
