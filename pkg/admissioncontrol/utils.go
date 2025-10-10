package admissioncontrol

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
)

// SensorEventToAdmCtrlReq converts a sensor event request into a admission control request.
func SensorEventToAdmCtrlReq(event *central.SensorEvent) (*sensor.AdmCtrlUpdateResourceRequest, error) {
	switch res := event.GetResource().(type) {
	case *central.SensorEvent_Synced:
		syncedResource := &sensor.AdmCtrlUpdateResourceRequest_Synced{
			Synced: &sensor.AdmCtrlUpdateResourceRequest_ResourcesSynced{},
		}
		return sensor.AdmCtrlUpdateResourceRequest_builder{
			Resource: syncedResource,
		}.Build(), nil
	case *central.SensorEvent_Pod:
		action := event.GetAction()
		podResource := &sensor.AdmCtrlUpdateResourceRequest_Pod{
			Pod: event.GetPod(),
		}
		return sensor.AdmCtrlUpdateResourceRequest_builder{
			Action:   &action,
			Resource: podResource,
		}.Build(), nil
	case *central.SensorEvent_Deployment:
		action := event.GetAction()
		deploymentResource := &sensor.AdmCtrlUpdateResourceRequest_Deployment{
			Deployment: event.GetDeployment(),
		}
		return sensor.AdmCtrlUpdateResourceRequest_builder{
			Action:   &action,
			Resource: deploymentResource,
		}.Build(), nil
	case *central.SensorEvent_Namespace:
		action := event.GetAction()
		namespaceResource := &sensor.AdmCtrlUpdateResourceRequest_Namespace{
			Namespace: event.GetNamespace(),
		}
		return sensor.AdmCtrlUpdateResourceRequest_builder{
			Action:   &action,
			Resource: namespaceResource,
		}.Build(), nil
	default:
		return nil, errors.Errorf("Cannot transform sensor event of type %T to admission control request message", res)
	}
}
