package dispatchers

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ResultDispatcher handles compliance check result objects
type ResultDispatcher struct{}

// NewResultDispatcher creates and returns a new compliance check result dispatcher.
func NewResultDispatcher() *ResultDispatcher {
	return &ResultDispatcher{}
}

func statusToProtoStatus(status string) storage.ComplianceOperatorCheckResult_CheckStatus {
	switch status {
	case checkResultPass:
		return storage.ComplianceOperatorCheckResult_PASS
	case checkResultFail:
		return storage.ComplianceOperatorCheckResult_FAIL
	case checkResultInfo:
		return storage.ComplianceOperatorCheckResult_INFO
	case checkResultManual:
		return storage.ComplianceOperatorCheckResult_MANUAL
	case checkResultError:
		return storage.ComplianceOperatorCheckResult_ERROR
	case checkResultNotApplicable:
		return storage.ComplianceOperatorCheckResult_NOT_APPLICABLE
	case checkResultInconsistent:
		return storage.ComplianceOperatorCheckResult_INCONSISTENT
	default:
		return storage.ComplianceOperatorCheckResult_UNSET
	}
}

func statusToV2Status(status string) central.ComplianceOperatorCheckResultV2_CheckStatus {
	switch status {
	case checkResultPass:
		return central.ComplianceOperatorCheckResultV2_PASS
	case checkResultFail:
		return central.ComplianceOperatorCheckResultV2_FAIL
	case checkResultInfo:
		return central.ComplianceOperatorCheckResultV2_INFO
	case checkResultManual:
		return central.ComplianceOperatorCheckResultV2_MANUAL
	case checkResultError:
		return central.ComplianceOperatorCheckResultV2_ERROR
	case checkResultNotApplicable:
		return central.ComplianceOperatorCheckResultV2_NOT_APPLICABLE
	case checkResultInconsistent:
		return central.ComplianceOperatorCheckResultV2_INCONSISTENT
	default:
		return central.ComplianceOperatorCheckResultV2_UNSET
	}
}

func getScanName(labels map[string]string) string {
	if value, ok := labels[complianceScanLabel]; ok {
		return value
	}

	return ""
}

func getSuiteName(labels map[string]string) string {
	if value, ok := labels[suiteLabel]; ok {
		return value
	}

	return ""
}

// ProcessEvent processes a compliance operator check result
func (c *ResultDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	unstructuredObject, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Not of type 'unstructured': %T", obj)
		return nil
	}

	id := string(unstructuredObject.GetUID())
	checkID, _, _ := unstructured.NestedString(unstructuredObject.Object, "id")
	status, _, _ := unstructured.NestedString(unstructuredObject.Object, "status")
	severity, _, _ := unstructured.NestedString(unstructuredObject.Object, "severity")
	description, _, _ := unstructured.NestedString(unstructuredObject.Object, "description")
	instructions, _, _ := unstructured.NestedString(unstructuredObject.Object, "instructions")
	rationale, _, _ := unstructured.NestedString(unstructuredObject.Object, "rationale")

	labels := unstructuredObject.GetLabels()
	annotations := unstructuredObject.GetAnnotations()

	events := []*central.SensorEvent{
		{
			Id:     id,
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorResult{
				ComplianceOperatorResult: &storage.ComplianceOperatorCheckResult{
					Id:           id,
					CheckId:      checkID,
					CheckName:    unstructuredObject.GetName(),
					Status:       statusToProtoStatus(status),
					Description:  description,
					Instructions: instructions,
					Labels:       labels,
					Annotations:  annotations,
				},
			},
		},
	}

	if centralcaps.Has(centralsensor.ComplianceV2Integrations) {
		warnings, _, _ := unstructured.NestedStringSlice(unstructuredObject.Object, "warnings")
		valuesUsed, _, _ := unstructured.NestedStringSlice(unstructuredObject.Object, "valuesUsed")

		creationTimestamp := unstructuredObject.GetCreationTimestamp()

		events = append(events, &central.SensorEvent{
			Id:     id,
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorResultV2{
				ComplianceOperatorResultV2: &central.ComplianceOperatorCheckResultV2{
					Id:           id,
					CheckId:      checkID,
					CheckName:    unstructuredObject.GetName(),
					Status:       statusToV2Status(status),
					Severity:     severityToV2Severity(severity),
					Description:  description,
					Instructions: instructions,
					Labels:       labels,
					Annotations:  annotations,
					CreatedTime:  protoconv.ConvertTimeToTimestamp(creationTimestamp.Time),
					ScanName:     getScanName(labels),
					SuiteName:    getSuiteName(labels),
					Rationale:    rationale,
					ValuesUsed:   valuesUsed,
					Warnings:     warnings,
				},
			},
		})
	}

	return component.NewEvent(events...)
}
