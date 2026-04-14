package dispatchers

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/set"
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
	// k8s name (not its XCCDF Status.ID) when the tailored profile contains custom rules
	// (annotation compliance.openshift.io/tailored-profile-contains-custom-rules=true, see
	// https://github.com/ComplianceAsCode/compliance-operator/blob/197c942793f0f0ef81ca39e4e9082271218b8b42/pkg/controller/scansettingbinding/scansettingbinding_controller.go#L555-L563
	// for details). We must use the same value as ProfileId so that BuildProfileRefID
	// produces matching UUIDs on both the profile and the scan sides.
	profileID := tailoredProfile.Status.ID
	if tailoredProfile.GetAnnotations()[v1alpha1.CustomRuleProfileAnnotation] == "true" {
		profileID = tailoredProfile.GetName()
	}

	protoProfile := &storage.ComplianceOperatorProfile{
		Id:          string(tailoredProfile.GetUID()),
		ProfileId:   profileID,
		Name:        tailoredProfile.GetName(),
		Labels:      tailoredProfile.GetLabels(),
		Annotations: tailoredProfile.GetAnnotations(),
		Description: tailoredProfile.Spec.Description,
	}

	removedRules := set.NewStringSet()
	for _, rule := range tailoredProfile.Spec.DisableRules {
		removedRules.Add(rule.Name)
	}

	for _, r := range baseProfile.Rules {
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

	if centralcaps.Has(centralsensor.ComplianceV2TailoredProfiles) {
		protoProfileV2 := &central.ComplianceOperatorProfileV2{
			Id:           protoProfile.GetId(),
			ProfileId:    protoProfile.GetProfileId(),
			Name:         protoProfile.GetName(),
			Labels:       protoProfile.GetLabels(),
			Annotations:  protoProfile.GetAnnotations(),
			Description:  protoProfile.GetDescription(),
			Title:        tailoredProfile.Spec.Title,
			OperatorKind: central.ComplianceOperatorProfileV2_TAILORED_PROFILE,
		}

		var ruleNames []string
		for _, rule := range protoProfile.GetRules() {
			protoProfileV2.Rules = append(protoProfileV2.Rules, &central.ComplianceOperatorProfileV2_Rule{RuleName: rule.GetName()})
			ruleNames = append(ruleNames, rule.GetName())
		}

		protoProfileV2.EquivalenceHash = computeProfileEquivalenceHash(
			tailoredProfile.GetName(),
			tailoredProfile.GetNamespace(),
			tailoredProfile.Spec.Description,
			tailoredProfile.Spec.Title,
			ruleNames,
			tailoredProfile.Spec.SetValues,
		)

		events = append(events, &central.SensorEvent{
			Id:     protoProfileV2.GetId(),
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorProfileV2{
				ComplianceOperatorProfileV2: protoProfileV2,
			},
		})
	}

	return component.NewEvent(events...)
}

// computeProfileEquivalenceHash returns a SHA-256 hex digest that identifies whether two
// profiles with the same name carry equivalent content. Fields are NUL-separated to prevent
// collisions between adjacent values; rule names and setValues entries are sorted for order
// independence.
//
// IMPORTANT — rolling-upgrade impact: changing the inputs or serialization of this function
// changes the hash for every tailored profile. During a rolling sensor upgrade, clusters
// running different sensor versions will produce different hashes for the same TP, causing
// those TPs to be filtered out of the multi-cluster profile picker and rejected from scan
// config creation/update until all sensors converge to the new version.
// Remediation: set ROX_COMPLIANCE_SKIP_TAILORED_PROFILE_EQUIVALENCE_CHECK=true on Central
// while sensor versions are mixed, then disable it once all sensors are upgraded.
func computeProfileEquivalenceHash(name, namespace, description, title string, ruleNames []string, setValues []v1alpha1.VariableValueSpec) string {
	sortedRules := make([]string, len(ruleNames))
	copy(sortedRules, ruleNames)
	slices.Sort(sortedRules)

	sortedVals := slices.Clone(setValues)
	slices.SortFunc(sortedVals, func(a, b v1alpha1.VariableValueSpec) int {
		if c := strings.Compare(a.Name, b.Name); c != 0 {
			return c
		}
		if c := strings.Compare(a.Rationale, b.Rationale); c != 0 {
			return c
		}
		return strings.Compare(a.Value, b.Value)
	})

	h := sha256.New()
	for _, s := range []string{name, namespace, description, title} {
		_, _ = h.Write([]byte(s))
		_, _ = h.Write([]byte{0})
	}
	for _, r := range sortedRules {
		_, _ = h.Write([]byte(r))
		_, _ = h.Write([]byte{0})
	}
	for _, v := range sortedVals {
		for _, s := range []string{v.Name, v.Rationale, v.Value} {
			_, _ = h.Write([]byte(s))
			_, _ = h.Write([]byte{0})
		}
	}
	hash := hex.EncodeToString(h.Sum(nil))
	// Example output:
	//
	//   Tailored profile "ocp4-cis-custom" equivalence hash computed with the following fields:
	//     name        = "ocp4-cis-custom"
	//     namespace   = "openshift-compliance"
	//     title       = "CIS OpenShift 4 Benchmark (tailored)"
	//     description = "My custom tailored profile"
	//     rules (3)   =
	//       - api_server_anonymous_auth
	//       - api_server_audit_log_maxage
	//       - api_server_audit_log_maxbackup
	//     setValues (1) =
	//       - var-some-name=value (rationale: because)
	//   Resulting hash: a3f7c2d9e1b4f8a06c5d2e7b3f9a1c4d8e2b5f7a0c3d6e9b2f5a8c1d4e7b0f3
	setValLines := make([]string, 0, len(sortedVals))
	for _, v := range sortedVals {
		setValLines = append(setValLines, v.Name+"="+v.Value+" (rationale: "+v.Rationale+")")
	}
	log.Debugf("Tailored profile %q equivalence hash computed with the following fields:"+
		"\n  name          = %q"+
		"\n  namespace     = %q"+
		"\n  title         = %q"+
		"\n  description   = %q"+
		"\n  rules (%d)    =\n    - %s"+
		"\n  setValues (%d) =\n    - %s"+
		"\nResulting hash: %s",
		name, name, namespace, title, description,
		len(sortedRules), strings.Join(sortedRules, "\n    - "),
		len(sortedVals), strings.Join(setValLines, "\n    - "), hash)
	return hash
}
