package dispatchers

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

// TailoredProfileDispatcher handles compliance operator tailored profile objects
type TailoredProfileDispatcher struct {
	profileLister cache.GenericLister
}

// NewTailoredProfileDispatcher creates and returns a new tailored profile dispatcher
func NewTailoredProfileDispatcher(profileLister cache.GenericLister) *TailoredProfileDispatcher {
	return &TailoredProfileDispatcher{
		profileLister: profileLister,
	}
}

// ProcessEvent processes a compliance operator tailored profile
func (c *TailoredProfileDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	var tailoredProfile v1alpha1.TailoredProfile

	unstructuredObject, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Not of type 'unstructured': %T", obj)
		return nil
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, &tailoredProfile); err != nil {
		log.Errorf("error converting unstructured to tailored compliance profile: %v", err)
		return nil
	}

	if tailoredProfile.Status.ID == "" {
		log.Warnf("Tailored profile %s does not have an ID. Skipping...", tailoredProfile.Name)
		return nil
	}

	var baseProfile v1alpha1.Profile
	if tailoredProfile.Spec.Extends != "" {
		profileObj, err := c.profileLister.ByNamespace(tailoredProfile.GetNamespace()).Get(tailoredProfile.Spec.Extends)
		if err != nil {
			log.Errorf("error getting profile %s: %v", tailoredProfile.Spec.Extends, err)
			return nil
		}
		unstructuredObject, ok = profileObj.(*unstructured.Unstructured)
		if !ok {
			log.Errorf("Fetched profile not of type 'unstructured': %T", profileObj)
			return nil
		}

		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, &baseProfile); err != nil {
			log.Errorf("error converting unstructured to compliance profile: %v", err)
			return nil
		}
	}

	// The compliance operator sets ComplianceScan.Spec.Profile to the tailored profile's
	// k8s name (not its XCCDF Status.ID) when the tailored profile contains custom rules.
	profileID := tailoredProfile.Status.ID
	if tailoredProfile.GetAnnotations()[v1alpha1.CustomRuleProfileAnnotation] == "true" {
		profileID = tailoredProfile.GetName()
	}

	uid := string(tailoredProfile.GetUID())

	removedRules := set.NewStringSet()
	for _, rule := range tailoredProfile.Spec.DisableRules {
		removedRules.Add(rule.Name)
	}

	var rules []*central.ComplianceOperatorProfileV2_Rule
	for _, r := range baseProfile.Rules {
		if removedRules.Contains(string(r)) {
			continue
		}
		rules = append(rules, &central.ComplianceOperatorProfileV2_Rule{RuleName: string(r)})
	}
	for _, rule := range tailoredProfile.Spec.EnableRules {
		rules = append(rules, &central.ComplianceOperatorProfileV2_Rule{RuleName: rule.Name})
	}

	return component.NewEvent(&central.SensorEvent{
		Id:     uid,
		Action: action,
		Resource: &central.SensorEvent_ComplianceOperatorProfileV2{
			ComplianceOperatorProfileV2: &central.ComplianceOperatorProfileV2{
				Id:           uid,
				ProfileId:    profileID,
				Name:         tailoredProfile.GetName(),
				Labels:       tailoredProfile.GetLabels(),
				Annotations:  tailoredProfile.GetAnnotations(),
				Description:  tailoredProfile.Spec.Description,
				Title:        tailoredProfile.Spec.Title,
				OperatorKind: central.ComplianceOperatorProfileV2_TAILORED_PROFILE,
				Rules:        rules,
			},
		},
	})
}
