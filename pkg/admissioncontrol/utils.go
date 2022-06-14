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
		return &sensor.AdmCtrlUpdateResourceRequest{
			Resource: &sensor.AdmCtrlUpdateResourceRequest_Synced{
				Synced: &sensor.AdmCtrlUpdateResourceRequest_ResourcesSynced{},
			},
		}, nil
	case *central.SensorEvent_Pod:
		return &sensor.AdmCtrlUpdateResourceRequest{
			Action: event.GetAction(),
			Resource: &sensor.AdmCtrlUpdateResourceRequest_Pod{
				Pod: event.GetPod(),
			},
		}, nil
	case *central.SensorEvent_Deployment:
		return &sensor.AdmCtrlUpdateResourceRequest{
			Action: event.GetAction(),
			Resource: &sensor.AdmCtrlUpdateResourceRequest_Deployment{
				Deployment: event.GetDeployment(),
			},
		}, nil
	case *central.SensorEvent_Namespace:
		return &sensor.AdmCtrlUpdateResourceRequest{
			Action: event.GetAction(),
			Resource: &sensor.AdmCtrlUpdateResourceRequest_Namespace{
				Namespace: event.GetNamespace(),
			},
		}, nil
	default:
		return nil, errors.Errorf("Cannot transform sensor event of type %T to admission control request message", res)
	}
}
