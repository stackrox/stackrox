package admissioncontrol

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"google.golang.org/protobuf/proto"
)

// SensorEventToAdmCtrlReq converts a sensor event request into a admission control request.
func SensorEventToAdmCtrlReq(event *central.SensorEvent) (*sensor.AdmCtrlUpdateResourceRequest, error) {
	switch res := event.WhichResource(); res {
	case central.SensorEvent_Synced_case:
		acurr := &sensor.AdmCtrlUpdateResourceRequest{}
		acurr.SetSynced(&sensor.AdmCtrlUpdateResourceRequest_ResourcesSynced{})
		return acurr, nil
	case central.SensorEvent_Pod_case:
		acurr := &sensor.AdmCtrlUpdateResourceRequest{}
		acurr.SetAction(event.GetAction())
		acurr.SetPod(proto.ValueOrDefault(event.GetPod()))
		return acurr, nil
	case central.SensorEvent_Deployment_case:
		acurr := &sensor.AdmCtrlUpdateResourceRequest{}
		acurr.SetAction(event.GetAction())
		acurr.SetDeployment(proto.ValueOrDefault(event.GetDeployment()))
		return acurr, nil
	case central.SensorEvent_Namespace_case:
		acurr := &sensor.AdmCtrlUpdateResourceRequest{}
		acurr.SetAction(event.GetAction())
		acurr.SetNamespace(proto.ValueOrDefault(event.GetNamespace()))
		return acurr, nil
	default:
		return nil, errors.Errorf("Cannot transform sensor event of type %v to admission control request message", res)
	}
}
