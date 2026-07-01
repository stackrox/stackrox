package dispatchers

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// ScanDispatcher handles compliance operator scan objects
type ScanDispatcher struct {
}

// NewScanDispatcher creates and returns a new scan dispatcher
func NewScanDispatcher() *ScanDispatcher {
	return &ScanDispatcher{}
}

// ProcessEvent processes a compliance operator scan
func (c *ScanDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	var complianceScan v1alpha1.ComplianceScan

	unstructuredObject, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Not of type 'unstructured': %T", obj)
		return nil
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, &complianceScan); err != nil {
		log.Errorf("error converting unstructured to compliance scan: %v", err)
		return nil
	}

	uid := string(complianceScan.UID)

	creationTime, err := protocompat.ConvertTimeToTimestampOrError(complianceScan.CreationTimestamp.Time)
	if err != nil {
		log.Warnf("unable to convert creation time %v", err)
	}

	protoStatus := &central.ComplianceOperatorScanStatusV2{
		Phase:            string(complianceScan.Status.Phase),
		Result:           string(complianceScan.Status.Result),
		ErrorMessage:     complianceScan.Status.ErrorMessage,
		CurrentIndex:     complianceScan.Status.CurrentIndex,
		Warnings:         complianceScan.Status.Warnings,
		RemainingRetries: int64(complianceScan.Status.RemainingRetries),
		StartTime:        creationTime,
	}

	if complianceScan.Status.EndTimestamp != nil {
		endTime, err := protocompat.ConvertTimeToTimestampOrError(complianceScan.Status.EndTimestamp.Time)
		if err != nil {
			log.Warnf("unable to convert end time %v", err)
		} else {
			protoStatus.EndTime = endTime
		}
	}

	if complianceScan.Status.StartTimestamp != nil {
		startTime, err := protocompat.ConvertTimeToTimestampOrError(complianceScan.Status.StartTimestamp.Time)
		if err != nil {
			log.Warnf("unable to convert start time %v", err)
		} else {
			protoStatus.LastStartTime = startTime
		}
	}

	protoScan := &central.ComplianceOperatorScanV2{
		Id:          uid,
		Name:        complianceScan.Name,
		ProfileId:   complianceScan.Spec.Profile,
		Labels:      complianceScan.Labels,
		Annotations: complianceScan.Annotations,
		ScanType:    string(complianceScan.Spec.ScanType),
		Status:      protoStatus,
	}

	return component.NewEvent(&central.SensorEvent{
		Id:     protoScan.GetId(),
		Action: action,
		Resource: &central.SensorEvent_ComplianceOperatorScanV2{
			ComplianceOperatorScanV2: protoScan,
		},
	})
}
