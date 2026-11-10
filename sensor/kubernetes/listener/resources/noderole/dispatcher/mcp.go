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

// MachineConfigPoolDispatcher handles MachineConfigPool events and updates the MachineConfigPool store.
type MachineConfigPoolDispatcher struct {
	store     *store.MachineConfigPoolStore
	publisher publisher
}

// NewMachineConfigPoolDispatcher creates a new MachineConfigPool dispatcher.
func NewMachineConfigPoolDispatcher(store *store.MachineConfigPoolStore, publisher publisher) *MachineConfigPoolDispatcher {
	return &MachineConfigPoolDispatcher{store: store, publisher: publisher}
}

// ProcessEvent processes a MachineConfigPool event.
func (d *MachineConfigPoolDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
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
	} else {
		changed = d.store.Add(name)
	}

	if changed && d.publisher != nil {
		if err := d.publisher.Publish(&events.NodeRoleChangeEvent{}); err != nil {
			log.Errorf("Failed to publish NodeRoleChangeEvent for MachineConfigPool %q: %v", name, err)
		}
	}
	// MCP state is propagated via pubsub, not through the resource event pipeline.
	return nil
}
