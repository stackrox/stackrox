package filesystem

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	v1 "k8s.io/api/core/v1"
)

type SettingsManager interface {
	UpdateFactSettings(policies []*storage.Policy)

	ConfigMapStream() concurrency.ReadOnlyValueStream[*v1.ConfigMap]
}
