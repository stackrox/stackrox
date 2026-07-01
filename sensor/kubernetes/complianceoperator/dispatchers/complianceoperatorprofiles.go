package dispatchers

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	log = logging.LoggerForModule()
)

// ProfileDispatcher handles compliance operator profile objects
type ProfileDispatcher struct {
}

// NewProfileDispatcher creates and returns a new profile dispatcher
func NewProfileDispatcher() *ProfileDispatcher {
	return &ProfileDispatcher{}
}

// ProcessEvent processes a compliance operator profile
func (c *ProfileDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	var complianceProfile v1alpha1.Profile

	unstructuredObject, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Not of type 'unstructured': %T", obj)
		return nil
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, &complianceProfile); err != nil {
		log.Errorf("error converting unstructured to compliance profile: %v", err)
		return nil
	}

	uid := string(complianceProfile.UID)

	protoProfile := &central.ComplianceOperatorProfileV2{
		Id:             uid,
		ProfileId:      complianceProfile.ID,
		Name:           complianceProfile.Name,
		ProfileVersion: complianceProfile.Version,
		Labels:         complianceProfile.Labels,
		Annotations:    complianceProfile.Annotations,
		Description:    complianceProfile.Description,
		Title:          complianceProfile.Title,
		OperatorKind:   central.ComplianceOperatorProfileV2_PROFILE,
	}

	for _, r := range complianceProfile.Rules {
		protoProfile.Rules = append(protoProfile.Rules, &central.ComplianceOperatorProfileV2_Rule{
			RuleName: string(r),
		})
	}

	for _, v := range complianceProfile.Values {
		protoProfile.Values = append(protoProfile.Values, string(v))
	}

	return component.NewEvent(&central.SensorEvent{
		Id:     uid,
		Action: action,
		Resource: &central.SensorEvent_ComplianceOperatorProfileV2{
			ComplianceOperatorProfileV2: protoProfile,
		},
	})
}
