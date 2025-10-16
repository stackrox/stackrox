package dispatchers

import (
	"strconv"
	"strings"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"google.golang.org/protobuf/proto"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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

	remediation := &v1alpha1.ComplianceRemediation{}

	unstructuredObject, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Not of type 'unstructured': %T", obj)
		return nil
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, remediation); err != nil {
		log.Errorf("error converting unstructured to compliance remediation: %v", err)
		return nil
	}

	// For nextgen compliance this ID is used to tell us which clusters have which remediations.  It is also
	// useful for the deduping from sensor.
	uid := string(remediation.UID)

	checkResultName := removeSuffix(remediation.GetName())

	var currentObjJSON []byte
	var err error
	if remediation.Spec.Current.Object != nil {
		currentObjJSON, err = remediation.Spec.Current.Object.MarshalJSON()
		if err != nil {
			log.Error("Error marshalling current obj from remediation", err)
			return nil
		}
	}

	var outdatedObjJSON []byte
	if remediation.Spec.Outdated.Object != nil {
		outdatedObjJSON, err = remediation.Spec.Outdated.Object.MarshalJSON()
		if err != nil {
			log.Error("Error marshalling outdated obj from remediation", err)
			return nil
		}
	}

	remediationCentral := &central.ComplianceOperatorRemediationV2{}
	remediationCentral.SetId(uid)
	remediationCentral.SetName(remediation.Name)
	remediationCentral.SetApply(remediation.IsApplied())
	remediationCentral.SetComplianceCheckResultName(checkResultName)
	remediationCentral.SetCurrentObject(string(currentObjJSON))
	remediationCentral.SetOutdatedObject(string(outdatedObjJSON))
	remediationCentral.SetEnforcementType(remediation.GetEnforcementType())

	se := &central.SensorEvent{}
	se.SetId(uid)
	se.SetAction(action)
	se.SetComplianceOperatorRemediationV2(proto.ValueOrDefault(remediationCentral))
	events := []*central.SensorEvent{
		se,
	}

	return component.NewEvent(events...)
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
