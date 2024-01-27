package dispatchers

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/sensor/common/centralcaps"
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

	// useful for the deduping from sensor.
	uid := string(complianceScan.UID)

	// We probably could have gotten away with re-using the storage proto here for the time being.
	// But we have a new field coming on for profiles and using the storage object even in an internal api
	// is a bad practice, so we will make that split now.  V1 and V2 compliance will both need to work for a period
	// of time.  However, we should not need to send the same profile twice, the pipeline can convert the V2 sensor message
	// so V1 and V2 objects can both be stored.

	protoScan := &storage.ComplianceOperatorScan{
		Id:          uid,
		Name:        complianceScan.Name,
		ProfileId:   complianceScan.Spec.Profile,
		Labels:      complianceScan.Labels,
		Annotations: complianceScan.Annotations,
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
		startTime, err := types.TimestampProto(complianceScan.CreationTimestamp.Time)
		if err != nil {
			log.Warnf("unable to convert start time %v", err)
		}

		var endTime *types.Timestamp
		if complianceScan.Status.EndTimestamp != nil {
			endTime, err = types.TimestampProto(complianceScan.Status.EndTimestamp.Time)
			if err != nil {
				log.Warnf("unable to convert end time %v", err)
			}
		}

		protoStatus := &central.ComplianceOperatorScanStatusV2{
			Phase:            string(complianceScan.Status.Phase),
			Result:           string(complianceScan.Status.Result),
			ErrorMessage:     complianceScan.Status.ErrorMessage,
			CurrentIndex:     complianceScan.Status.CurrentIndex,
			Warnings:         complianceScan.Status.Warnings,
			RemainingRetries: int64(complianceScan.Status.RemainingRetries),
			StartTime:        startTime,
			EndTime:          endTime,
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
		events = append(events, &central.SensorEvent{
			Id:     protoScan.GetId(),
			Action: action,
			Resource: &central.SensorEvent_ComplianceOperatorScanV2{
				ComplianceOperatorScanV2: protoScan,
			},
		})
	}

	return component.NewEvent(events...)
}
