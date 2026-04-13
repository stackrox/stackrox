package dispatchers

import (
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	unstructuredObject, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Not of type 'unstructured': %T", obj)
		return nil
	}

	// useful for the deduping from sensor.
	uid := string(unstructuredObject.GetUID())

	specProfile, _, _ := unstructured.NestedString(unstructuredObject.Object, "spec", "profile")

	// We probably could have gotten away with re-using the storage proto here for the time being.
	// But we have a new field coming on for profiles and using the storage object even in an internal api
	// is a bad practice, so we will make that split now.  V1 and V2 compliance will both need to work for a period
	// of time.  However, we should not need to send the same profile twice, the pipeline can convert the V2 sensor message
	// so V1 and V2 objects can both be stored.

	protoScan := &storage.ComplianceOperatorScan{
		Id:          uid,
		Name:        unstructuredObject.GetName(),
		ProfileId:   specProfile,
		Labels:      unstructuredObject.GetLabels(),
		Annotations: unstructuredObject.GetAnnotations(),
	}
	events := []*central.SensorEvent{
		{
			Id:     protoScan.GetId(),
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorScan{
				ComplianceOperatorScan: protoScan,
			},
		},
	}

	// Build a V2 event if central is capable of receiving it
	if centralcaps.Has(centralsensor.ComplianceV2Integrations) {
		creationTime, err := protocompat.ConvertTimeToTimestampOrError(unstructuredObject.GetCreationTimestamp().Time)
		if err != nil {
			log.Warnf("unable to convert creation time %v", err)
		}

		status, _ := unstructuredObject.Object["status"].(map[string]interface{})
		statusPhase, _ := status["phase"].(string)
		statusResult, _ := status["result"].(string)
		statusErrorMsg, _ := status["errormsg"].(string)
		statusCurrentIndex, _, _ := unstructured.NestedInt64(status, "currentIndex")
		statusWarnings, _ := status["warnings"].(string)
		statusRemainingRetries, _, _ := unstructured.NestedInt64(status, "remainingRetries")

		protoStatus := &central.ComplianceOperatorScanStatusV2{
			Phase:            statusPhase,
			Result:           statusResult,
			ErrorMessage:     statusErrorMsg,
			CurrentIndex:     statusCurrentIndex,
			Warnings:         statusWarnings,
			RemainingRetries: statusRemainingRetries,
			StartTime:        creationTime,
		}

		if endTimestampStr, ok, _ := unstructured.NestedString(status, "endTimestamp"); ok && endTimestampStr != "" {
			if endTime, err := time.Parse(time.RFC3339, endTimestampStr); err == nil {
				if ts, err := protocompat.ConvertTimeToTimestampOrError(endTime); err == nil {
					protoStatus.EndTime = ts
				} else {
					log.Warnf("unable to convert end time %v", err)
				}
			}
		}

		if startTimestampStr, ok, _ := unstructured.NestedString(status, "startTimestamp"); ok && startTimestampStr != "" {
			if startTime, err := time.Parse(time.RFC3339, startTimestampStr); err == nil {
				if ts, err := protocompat.ConvertTimeToTimestampOrError(startTime); err == nil {
					protoStatus.LastStartTime = ts
				} else {
					log.Warnf("unable to convert start time %v", err)
				}
			}
		}

		scanType, _, _ := unstructured.NestedString(unstructuredObject.Object, "spec", "scanType")

		protoScanV2 := &central.ComplianceOperatorScanV2{
			Id:          uid,
			Name:        unstructuredObject.GetName(),
			ProfileId:   specProfile,
			Labels:      unstructuredObject.GetLabels(),
			Annotations: unstructuredObject.GetAnnotations(),
			ScanType:    scanType,
			Status:      protoStatus,
		}
		events = append(events, &central.SensorEvent{
			Id:     protoScanV2.GetId(),
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorScanV2{
				ComplianceOperatorScanV2: protoScanV2,
			},
		})
	}

	return component.NewEvent(events...)
}
