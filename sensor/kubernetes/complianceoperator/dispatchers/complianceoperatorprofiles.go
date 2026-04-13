package dispatchers

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	unstructuredObject, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Not of type 'unstructured': %T", obj)
		return nil
	}

	// For nextgen compliance this ID is used to tell us which clusters have which profiles.  It is also
	// useful for the deduping from sensor.
	uid := string(unstructuredObject.GetUID())

	profileID, _, _ := unstructured.NestedString(unstructuredObject.Object, "id")
	description, _, _ := unstructured.NestedString(unstructuredObject.Object, "description")

	protoProfile := &storage.ComplianceOperatorProfile{
		Id:          uid,
		ProfileId:   profileID,
		Name:        unstructuredObject.GetName(),
		Labels:      unstructuredObject.GetLabels(),
		Annotations: unstructuredObject.GetAnnotations(),
		Description: description,
	}

	rulesSlice, _, _ := unstructured.NestedStringSlice(unstructuredObject.Object, "rules")
	for _, r := range rulesSlice {
		protoProfile.Rules = append(protoProfile.Rules, &storage.ComplianceOperatorProfile_Rule{
			Name: r,
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

	if centralcaps.Has(centralsensor.ComplianceV2Integrations) {
		title, _, _ := unstructured.NestedString(unstructuredObject.Object, "title")
		version, _, _ := unstructured.NestedString(unstructuredObject.Object, "version")

		protoProfileV2 := &central.ComplianceOperatorProfileV2{
			Id:             uid,
			ProfileId:      profileID,
			Name:           unstructuredObject.GetName(),
			ProfileVersion: version,
			Labels:         unstructuredObject.GetLabels(),
			Annotations:    unstructuredObject.GetAnnotations(),
			Description:    description,
			Title:          title,
			OperatorKind:   central.ComplianceOperatorProfileV2_PROFILE,
		}

		for _, r := range rulesSlice {
			protoProfileV2.Rules = append(protoProfileV2.Rules, &central.ComplianceOperatorProfileV2_Rule{
				RuleName: r,
			})
		}

		valuesSlice, _, _ := unstructured.NestedStringSlice(unstructuredObject.Object, "values")
		for _, v := range valuesSlice {
			protoProfileV2.Values = append(protoProfileV2.Values, v)
		}

		events = append(events, &central.SensorEvent{
			Id:     uid,
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorProfileV2{
				ComplianceOperatorProfileV2: protoProfileV2,
			},
		})
	}

	return component.NewEvent(events...)
}
