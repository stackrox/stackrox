package dispatchers

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"google.golang.org/protobuf/proto"
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

	protoProfile := &storage.ComplianceOperatorProfile{}
	protoProfile.SetId(uid)
	protoProfile.SetProfileId(complianceProfile.ID)
	protoProfile.SetName(complianceProfile.Name)
	protoProfile.SetLabels(complianceProfile.Labels)
	protoProfile.SetAnnotations(complianceProfile.Annotations)
	protoProfile.SetDescription(complianceProfile.Description)
	for _, r := range complianceProfile.Rules {
		cr := &storage.ComplianceOperatorProfile_Rule{}
		cr.SetName(string(r))
		protoProfile.SetRules(append(protoProfile.GetRules(), cr))
	}

	se := &central.SensorEvent{}
	se.SetId(uid)
	se.SetAction(action)
	se.SetComplianceOperatorProfile(proto.ValueOrDefault(protoProfile))
	events := []*central.SensorEvent{
		se,
	}

	if centralcaps.Has(centralsensor.ComplianceV2Integrations) {
		protoProfile := &central.ComplianceOperatorProfileV2{}
		protoProfile.SetId(uid)
		protoProfile.SetProfileId(complianceProfile.ID)
		protoProfile.SetName(complianceProfile.Name)
		protoProfile.SetProfileVersion(complianceProfile.Version)
		protoProfile.SetLabels(complianceProfile.Labels)
		protoProfile.SetAnnotations(complianceProfile.Annotations)
		protoProfile.SetDescription(complianceProfile.Description)
		protoProfile.SetTitle(complianceProfile.Title)

		for _, r := range complianceProfile.Rules {
			cr := &central.ComplianceOperatorProfileV2_Rule{}
			cr.SetRuleName(string(r))
			protoProfile.SetRules(append(protoProfile.GetRules(), cr))
		}

		for _, v := range complianceProfile.Values {
			protoProfile.SetValues(append(protoProfile.GetValues(), string(v)))
		}

		se2 := &central.SensorEvent{}
		se2.SetId(uid)
		se2.SetAction(action)
		se2.SetComplianceOperatorProfileV2(proto.ValueOrDefault(protoProfile))
		events = append(events, se2)
	}

	return component.NewEvent(events...)
}
