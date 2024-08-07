package notifier

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/expiringcache"
)

// AlertSet is a layer over an expiring cache specifically for alerts.
type AlertSet interface {
	Add(alert *storage.Alert)
	Remove(id string)
	GetAll() []*storage.Alert
}

// NewAlertSet returns a new AlertSet instance
func NewAlertSet(retryAlertsFor time.Duration) AlertSet {
	return &alertSetImpl{
		alerts: expiringcache.NewExpiringCache[string, *storage.Alert](retryAlertsFor),
	}
}

type alertSetImpl struct {
	alerts expiringcache.Cache[string, *storage.Alert]
}

func (as *alertSetImpl) Add(alert *storage.Alert) {
	as.alerts.Add(alert.GetId(), alert.CloneVT())
}

func (as *alertSetImpl) Remove(id string) {
	as.alerts.Remove(id)
}

func (as *alertSetImpl) GetAll() []*storage.Alert {
	return as.alerts.GetAll()
}
