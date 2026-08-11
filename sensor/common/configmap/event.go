package configmap

import (
	"github.com/stackrox/rox/sensor/common/pubsub"
	v1 "k8s.io/api/core/v1"
)

// ConfigMapUpdatedEvent carries a freshly computed ConfigMap to be persisted by a configMapPersister.
type ConfigMapUpdatedEvent struct {
	ConfigMap *v1.ConfigMap
}

// Topic implements pubsub.Event.
func (e *ConfigMapUpdatedEvent) Topic() pubsub.Topic {
	return pubsub.AdmCtrlConfigMapTopic
}

// Lane implements pubsub.Event.
func (e *ConfigMapUpdatedEvent) Lane() pubsub.LaneID {
	return pubsub.AdmCtrlConfigMapLane
}
