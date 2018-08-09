package listeners

import (
	"reflect"

	"github.com/stackrox/rox/generated/api/v1"
)

// EventWrap contains a Deployment and the original deployment event.
type EventWrap struct {
	*v1.SensorEvent
	OriginalSpec interface{}
}

func equalDeployments(d1, d2 *v1.Deployment) bool {
	// Save the values because we need to overwrite them for DeepEqual to ignore
	// inconsequential updates
	tempUpdatedAt := d1.UpdatedAt
	tempVersion := d1.Version

	d1.UpdatedAt = d2.UpdatedAt
	d1.Version = d2.Version
	equal := reflect.DeepEqual(d1, d2)

	d1.UpdatedAt = tempUpdatedAt
	d1.Version = tempVersion
	return equal
}

// Equals handles the comparisons between different event types by using deep equals
func (ew *EventWrap) Equals(newEW *EventWrap) bool {
	switch x := newEW.Resource.(type) {
	case *v1.SensorEvent_Deployment:
		return equalDeployments(newEW.GetDeployment(), ew.GetDeployment())
	case *v1.SensorEvent_NetworkPolicy:
		return reflect.DeepEqual(ew.GetNetworkPolicy(), newEW.GetNetworkPolicy())
	case *v1.SensorEvent_Namespace:
		return reflect.DeepEqual(ew.GetNamespace(), newEW.GetNamespace())
	case nil:
		logger.Errorf("Resource field is empty")
	default:
		logger.Errorf("No resource with type %T", x)
	}
	return false
}

// EventWrapResponse wraps the response from the server with the original object
type EventWrapResponse struct {
	*v1.SensorEventResponse
	OriginalSpec interface{}
}
