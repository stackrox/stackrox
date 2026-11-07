package dispatcher

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/events"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/noderole/store"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	log = logging.LoggerForModule()
)

type publisher interface {
	Publish(pubsub.Event) error
}

// Dispatcher handles MachineConfigPool events and updates the node-role store.
type Dispatcher struct {
	store     *store.Store
	publisher publisher
}

// New creates a new MachineConfigPool dispatcher.
func New(store *store.Store, publisher publisher) *Dispatcher {
	return &Dispatcher{store: store, publisher: publisher}
}

// ProcessEvent processes a MachineConfigPool event.
func (d *Dispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Expected *unstructured.Unstructured but got %T", obj)
		return nil
	}

	name := u.GetName()
	if name == "" {
		log.Errorf("MachineConfigPool has no name: %v", u.Object)
		return nil
	}

	var changed bool
	if action == central.ResourceAction_REMOVE_RESOURCE {
		changed = d.store.Remove(name)
		log.Debugf("Removed MachineConfigPool %q from node-role store (changed=%t)", name, changed)
	} else {
		changed = d.store.Add(name)
		log.Debugf("Added MachineConfigPool %q to node-role store (changed=%t)", name, changed)
	}

	if changed && d.publisher != nil {
		if err := d.publisher.Publish(&events.NodeRoleChangeEvent{}); err != nil {
			log.Errorf("Failed to publish NodeRoleChangeEvent for MachineConfigPool %q: %v", name, err)
		}
	}
	return nil
}
