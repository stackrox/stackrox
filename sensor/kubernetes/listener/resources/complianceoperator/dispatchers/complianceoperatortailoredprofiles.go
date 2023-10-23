package dispatchers

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/sensor/common/deduper"
	"github.com/stackrox/rox/sensor/common/store/reconciliation"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

// TailoredProfileDispatcher handles compliance operator tailored profile objects
type TailoredProfileDispatcher struct {
	profileLister       cache.GenericLister
	reconciliationStore reconciliation.Store
}

// NewTailoredProfileDispatcher creates and returns a new tailored profile dispatcher
func NewTailoredProfileDispatcher(store reconciliation.Store, profileLister cache.GenericLister) *TailoredProfileDispatcher {
	return &TailoredProfileDispatcher{
		profileLister:       profileLister,
		reconciliationStore: store,
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

	profileObj, err := c.profileLister.ByNamespace(tailoredProfile.GetNamespace()).Get(tailoredProfile.Spec.Extends)
	if err != nil {
		log.Errorf("error getting profile %s: %v", tailoredProfile.Spec.Extends, err)
		return nil
	}
	unstructuredObject, ok = profileObj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Fetched profile not of type 'unstructured': %T", obj)
		return nil
	}

	var complianceProfile v1alpha1.Profile
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, &complianceProfile); err != nil {
		log.Errorf("error converting unstructured to compliance profile: %v", err)
		return nil
	}

	protoProfile := &storage.ComplianceOperatorProfile{
		Id:        string(tailoredProfile.UID),
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

	events := []*central.SensorEvent{
		{
			Id:     protoProfile.GetId(),
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorProfile{
				ComplianceOperatorProfile: protoProfile,
			},
		},
	}
	if action == central.ResourceAction_REMOVE_RESOURCE {
		c.reconciliationStore.Remove(deduper.TypeComplianceOperatorProfile.String(), protoProfile.GetId())
	} else {
		c.reconciliationStore.Upsert(deduper.TypeComplianceOperatorProfile.String(), protoProfile.GetId())
	}
	return component.NewEvent(events...)
}
