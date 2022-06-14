package processor

import (
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/expiringcache"
)

const (
	// When we fail to notify on an alert, retry every hour for 4 hours, and only retry up to 100 alerts
	retryAlertsEvery = 5 * time.Minute
	retryAlertsFor   = 1 * time.Hour
)

// AlertSet is a layer over an expiring cache specifically for alerts.
type AlertSet interface {
	Add(alert *storage.Alert)
	Remove(id string)
	GetAll() []*storage.Alert
}

// NewAlertSet returns a new AlertSet instance
func NewAlertSet() AlertSet {
	return &alertSetImpl{
		alerts: expiringcache.NewExpiringCache(retryAlertsFor),
	}
}

type alertSetImpl struct {
	alerts expiringcache.Cache
}

func (as *alertSetImpl) Add(alert *storage.Alert) {
	as.alerts.Add(alert.GetId(), proto.Clone(alert))
}

func (as *alertSetImpl) Remove(id string) {
	as.alerts.Remove(id)
}

func (as *alertSetImpl) GetAll() []*storage.Alert {
	alertInterfaces := as.alerts.GetAll()

	ret := make([]*storage.Alert, 0, len(alertInterfaces))
	for _, ai := range alertInterfaces {
		ret = append(ret, ai.(*storage.Alert))
	}
	return ret
}
