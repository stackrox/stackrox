package dispatchers

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
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

	// For nextgen compliance this ID is used to tell us which clusters have which profiles.  It is also
	// useful for the deduping from sensor.
	uid := string(complianceProfile.UID)

	// We probably could have gotten away with re-using the storage proto here for the time being.
	// But we have a new field coming on for profiles and using the storage object even in an internal api
	// is a bad practice, so we will make that split now.  V1 and V2 compliance will both need to work for a period
	// of time.  However, we should not need to send the same profile twice, the pipeline can convert the V2 sensor message
	// so V1 and V2 objects can both be stored.
	if features.ComplianceEnhancements.Enabled() {
		protoProfile := &central.ComplianceOperatorProfileV2{
			Id:             uid,
			ProfileId:      complianceProfile.ID,
			Name:           complianceProfile.Name,
			ProfileVersion: "", // TODO(ROX-20536)
			Labels:         complianceProfile.Labels,
			Annotations:    complianceProfile.Annotations,
			Description:    complianceProfile.Description,
		}

		for _, r := range complianceProfile.Rules {
			protoProfile.Rules = append(protoProfile.Rules, &central.ComplianceOperatorProfileV2_Rule{
				RuleName: string(r),
			})
		}

		for _, v := range complianceProfile.Values {
			protoProfile.Values = append(protoProfile.Values, string(v))
		}

		events := []*central.SensorEvent{
			{
				Id:     uid,
				Action: action,
				Resource: &central.SensorEvent_ComplianceOperatorProfileV2{
					ComplianceOperatorProfileV2: protoProfile,
				},
			}}

		return component.NewEvent(events...)
	}

	protoProfile := &storage.ComplianceOperatorProfile{
		Id:          uid,
		ProfileId:   complianceProfile.ID,
		Name:        complianceProfile.Name,
		Labels:      complianceProfile.Labels,
		Annotations: complianceProfile.Annotations,
		Description: complianceProfile.Description,
	}
	for _, r := range complianceProfile.Rules {
		protoProfile.Rules = append(protoProfile.Rules, &storage.ComplianceOperatorProfile_Rule{
			Name: string(r),
		})
	}

	events := []*central.SensorEvent{
		{
			Id:     uid,
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorProfile{
				ComplianceOperatorProfile: protoProfile,
			},
		},
	}

	return component.NewEvent(events...)
}
