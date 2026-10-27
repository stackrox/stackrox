package dispatcher

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/noderole/store"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	log = logging.LoggerForModule()
)

// Dispatcher handles MachineConfigPool events and updates the node-role store.
type Dispatcher struct {
	store *store.Store
}

// New creates a new MachineConfigPool dispatcher.
func New(store *store.Store) *Dispatcher {
	return &Dispatcher{store: store}
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

	if action == central.ResourceAction_REMOVE_RESOURCE {
		d.store.Remove(name)
		log.Debugf("Removed MachineConfigPool %q from node-role store", name)
		return nil
	}

	matchLabels := extractMatchLabels(u)
	d.store.AddOrUpdate(name, matchLabels)
	// TODO: Send a message using the pubsub system (sensor/common/pubsub) if there's a change in the node roles where it is because it was deleted updated or created
	log.Debugf("Upserted MachineConfigPool %q with matchLabels %v", name, matchLabels)
	return nil
}

func extractMatchLabels(u *unstructured.Unstructured) map[string]string {
	nodeSelector, found, err := unstructured.NestedMap(u.Object, "spec", "nodeSelector", "matchLabels")
	if err != nil || !found {
		return nil
	}
	result := make(map[string]string, len(nodeSelector))
	for k, v := range nodeSelector {
		if s, ok := v.(string); ok {
			result[k] = s
		}
	}
	return result
}
