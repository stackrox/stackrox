package dispatcher

import (
	aiworkloadV1 "github.com/stackrox/rox/generated/internalapi/aiworkload/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func getStringField(obj map[string]interface{}, fields ...string) string {
	current := obj
	for i, field := range fields {
		if i == len(fields)-1 {
			val, _ := current[field].(string)
			return val
		}
		next, ok := current[field].(map[string]interface{})
		if !ok {
			return ""
		}
		current = next
	}
	return ""
}

func getResourceQuantity(obj map[string]interface{}, path []string, resourceName string) string {
	current := obj
	for _, field := range path {
		next, ok := current[field].(map[string]interface{})
		if !ok {
			return ""
		}
		current = next
	}
	val, _ := current[resourceName].(string)
	return val
}

func createEvent(u *unstructured.Unstructured, workload *aiworkloadV1.AIWorkload, action central.ResourceAction) *component.ResourceEvent {
	event := &central.SensorEvent{
		Id:     string(u.GetUID()),
		Action: action,
		Resource: &central.SensorEvent_AiWorkload{
			AiWorkload: workload,
		},
	}
	return component.NewEvent(event)
}
