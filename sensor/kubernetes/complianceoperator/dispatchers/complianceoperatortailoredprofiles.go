package dispatchers

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/sensor/common/centralcaps"
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

	uid := string(tailoredProfile.UID)
	var events []*central.SensorEvent

	// V1 events: need to fetch parent profile and compute effective rules
	// Only for extended TPs (from-scratch TPs don't have a parent profile for V1)
	if tailoredProfile.Spec.Extends != "" {
		profileObj, err := c.profileLister.ByNamespace(tailoredProfile.GetNamespace()).Get(tailoredProfile.Spec.Extends)
		if err != nil {
			log.Errorf("error getting profile %s: %v", tailoredProfile.Spec.Extends, err)
		} else {
			var complianceProfile v1alpha1.Profile
			profileUnstructured, ok := profileObj.(*unstructured.Unstructured)
			if !ok {
				log.Errorf("Fetched profile not of type 'unstructured': %T", profileObj)
			} else if err := runtime.DefaultUnstructuredConverter.FromUnstructured(profileUnstructured.Object, &complianceProfile); err != nil {
				log.Errorf("error converting unstructured to compliance profile: %v", err)
			} else {
				protoProfile := &storage.ComplianceOperatorProfile{
					Id:        uid,
					ProfileId: tailoredProfile.Status.ID,
					Name:      tailoredProfile.Name,
					// We want to use the original compliance profiles labels and annotations as they hold data about the type of profile
					Labels:      complianceProfile.Labels,
					Annotations: complianceProfile.Annotations,
					Description: stringutils.FirstNonEmpty(tailoredProfile.Spec.Description, complianceProfile.Description),
				}
				removedRules := set.NewStringSet()
				for _, rule := range tailoredProfile.Spec.DisableRules {
					removedRules.Add(rule.Name)
				}

				for _, r := range complianceProfile.Rules {
					if removedRules.Contains(string(r)) {
						continue
					}
					protoProfile.Rules = append(protoProfile.Rules, &storage.ComplianceOperatorProfile_Rule{
						Name: string(r),
					})
				}
				for _, rule := range tailoredProfile.Spec.EnableRules {
					protoProfile.Rules = append(protoProfile.Rules, &storage.ComplianceOperatorProfile_Rule{
						Name: rule.Name,
					})
				}

				events = append(events, &central.SensorEvent{
					Id:     uid,
					Action: action,
					Resource: &central.SensorEvent_ComplianceOperatorProfile{
						ComplianceOperatorProfile: protoProfile,
					},
				})
			}
		}
	}

	// V2 events: store the tailored profile with delta information
	if centralcaps.Has(centralsensor.ComplianceV2Integrations) {
		protoProfileV2 := &central.ComplianceOperatorProfileV2{
			Id:          uid,
			ProfileId:   tailoredProfile.Status.ID,
			Name:        tailoredProfile.Name,
			Labels:      tailoredProfile.Labels,
			Annotations: tailoredProfile.Annotations,
			Description: tailoredProfile.Spec.Description,
			Title:       tailoredProfile.Spec.Title,
			// TailoredDetails presence indicates this is a TailoredProfile
			TailoredDetails: &central.TailoredProfileDetails{
				Extends:      tailoredProfile.Spec.Extends,
				State:        string(tailoredProfile.Status.State),
				ErrorMessage: tailoredProfile.Status.ErrorMessage,
			},
		}

		// Populate disabled rules
		for _, rule := range tailoredProfile.Spec.DisableRules {
			protoProfileV2.TailoredDetails.DisabledRules = append(protoProfileV2.TailoredDetails.DisabledRules,
				&central.TailoredProfileRuleModification{
					Name:      rule.Name,
					Rationale: rule.Rationale,
				})
		}

		// Populate enabled rules
		for _, rule := range tailoredProfile.Spec.EnableRules {
			protoProfileV2.TailoredDetails.EnabledRules = append(protoProfileV2.TailoredDetails.EnabledRules,
				&central.TailoredProfileRuleModification{
					Name:      rule.Name,
					Rationale: rule.Rationale,
				})
		}

		// Populate manual rules
		for _, rule := range tailoredProfile.Spec.ManualRules {
			protoProfileV2.TailoredDetails.ManualRules = append(protoProfileV2.TailoredDetails.ManualRules,
				&central.TailoredProfileRuleModification{
					Name:      rule.Name,
					Rationale: rule.Rationale,
				})
		}

		// Populate set values
		for _, val := range tailoredProfile.Spec.SetValues {
			protoProfileV2.TailoredDetails.SetValues = append(protoProfileV2.TailoredDetails.SetValues,
				&central.TailoredProfileValueOverride{
					Name:      val.Name,
					Value:     val.Value,
					Rationale: val.Rationale,
				})
		}

		// For from-scratch TPs, populate rules from enableRules
		// For extended TPs, we store the delta only - scanner computes effective rules at runtime
		for _, rule := range tailoredProfile.Spec.EnableRules {
			protoProfileV2.Rules = append(protoProfileV2.Rules, &central.ComplianceOperatorProfileV2_Rule{
				RuleName: rule.Name,
			})
		}

		events = append(events, &central.SensorEvent{
			Id:     uid,
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorProfileV2{
				ComplianceOperatorProfileV2: protoProfileV2,
			},
		})
	}

	if len(events) == 0 {
		return nil
	}
	return component.NewEvent(events...)
}
