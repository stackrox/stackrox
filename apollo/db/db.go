package db

import (
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

// Storage is the interface for the persistent storage
type Storage interface {
	AddImage(image *v1.Image)
	RemoveImage(string)
	GetImages() []*v1.Image

	AddImageRule(*v1.ImageRule)
	RemoveImageRule(string)
	UpdateImageRule(*v1.ImageRule)
	GetImageRules() []*v1.ImageRule
	GetImageRule(string) *v1.ImageRule

	AddAlert(alert *v1.Alert)
	RemoveAlert(id string)
	GetAlert(id string) *v1.Alert
	GetAlerts() []*v1.Alert
}
