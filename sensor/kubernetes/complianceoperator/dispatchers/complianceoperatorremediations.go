package dispatchers

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// RemediationDispatcher handles compliance operator remediation objects
type RemediationDispatcher struct {
}

// NewRemediationDispatcher creates and returns a new remediation dispatcher
func NewRemediationDispatcher() *RemediationDispatcher {
	return &RemediationDispatcher{}
}

// ProcessEvent processes a compliance operator remediation
func (c *RemediationDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	if !centralcaps.Has(centralsensor.ComplianceV2Remediations) {
		return nil
	}

	unstructuredObject, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Not of type 'unstructured': %T", obj)
		return nil
	}

	// For nextgen compliance this ID is used to tell us which clusters have which remediations.  It is also
	// useful for the deduping from sensor.
	uid := string(unstructuredObject.GetUID())

	checkResultName := removeSuffix(unstructuredObject.GetName())

	spec, _ := unstructuredObject.Object["spec"].(map[string]interface{})

	apply, _, _ := unstructured.NestedBool(spec, "apply")

	var currentObjJSON []byte
	if currentMap, ok, _ := unstructured.NestedMap(spec, "current", "object"); ok && currentMap != nil {
		var err error
		currentObjJSON, err = json.Marshal(&unstructured.Unstructured{Object: currentMap})
		if err != nil {
			log.Error("Error marshalling current obj from remediation", err)
			return nil
		}
	}

	var outdatedObjJSON []byte
	if outdatedMap, ok, _ := unstructured.NestedMap(spec, "outdated", "object"); ok && outdatedMap != nil {
		var err error
		outdatedObjJSON, err = json.Marshal(&unstructured.Unstructured{Object: outdatedMap})
		if err != nil {
			log.Error("Error marshalling outdated obj from remediation", err)
			return nil
		}
	}

	status, _ := unstructuredObject.Object["status"].(map[string]interface{})
	applicationState, _ := status["applicationState"].(string)
	isApplied := isRemediationApplied(apply, applicationState)
	enforcementType := getEnforcementType(unstructuredObject.GetAnnotations())

	remediationCentral := &central.ComplianceOperatorRemediationV2{
		Id:                        uid,
		Name:                      unstructuredObject.GetName(),
		Apply:                     isApplied,
		ComplianceCheckResultName: checkResultName,
		CurrentObject:             string(currentObjJSON),
		OutdatedObject:            string(outdatedObjJSON),
		EnforcementType:           enforcementType,
	}

	events := []*central.SensorEvent{
		{
			Id:     uid,
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorRemediationV2{
				ComplianceOperatorRemediationV2: remediationCentral,
			},
		},
	}

	return component.NewEvent(events...)
}

// isRemediationApplied replicates the logic from v1alpha1.ComplianceRemediation.IsApplied().
func isRemediationApplied(apply bool, applicationState string) bool {
	applied := applicationState == remediationApplied
	outdatedButApplied := apply && applicationState == remediationOutdated
	appliedButUnmet := apply && applicationState == remediationMissingDependencies
	return applied || outdatedButApplied || appliedButUnmet
}

// getEnforcementType replicates the logic from v1alpha1.ComplianceRemediation.GetEnforcementType().
func getEnforcementType(annotations map[string]string) string {
	if len(annotations) == 0 {
		return "unknown"
	}
	etype, ok := annotations[remediationEnforcementTypeAnnotation]
	if !ok {
		return "unknown"
	}
	return etype
}

// Compliance remediations are name like the corresponding result it belongs to. There can be multiple remediations per
// result. The first remediation does not have a counter suffix, up from the second, all remediations have a suffix, e.g. `-1` or `-2`.
// This function removes the suffix to receive the result name.
func removeSuffix(name string) string {
	if !strings.Contains(name, "-") {
		return name
	}

	splitted := strings.Split(name, "-")
	lastIndex := len(splitted) - 1
	if _, err := strconv.Atoi(splitted[lastIndex]); err != nil {
		return name
	}

	return strings.Join(splitted[:len(splitted)-1], "-")
}
