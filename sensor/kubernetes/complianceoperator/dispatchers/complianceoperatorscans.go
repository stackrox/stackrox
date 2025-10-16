package dispatchers

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"google.golang.org/protobuf/proto"
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

	protoScan := &storage.ComplianceOperatorScan{}
	protoScan.SetId(uid)
	protoScan.SetName(complianceScan.Name)
	protoScan.SetProfileId(complianceScan.Spec.Profile)
	protoScan.SetLabels(complianceScan.Labels)
	protoScan.SetAnnotations(complianceScan.Annotations)
	se := &central.SensorEvent{}
	se.SetId(protoScan.GetId())
	se.SetAction(action)
	se.SetComplianceOperatorScan(proto.ValueOrDefault(protoScan))
	events := []*central.SensorEvent{
		se,
	}

	// Build a V2 event if central is capable of receiving it
	if centralcaps.Has(centralsensor.ComplianceV2Integrations) {
		creationTime, err := protocompat.ConvertTimeToTimestampOrError(complianceScan.CreationTimestamp.Time)
		if err != nil {
			log.Warnf("unable to convert creation time %v", err)
		}

		protoStatus := &central.ComplianceOperatorScanStatusV2{}
		protoStatus.SetPhase(string(complianceScan.Status.Phase))
		protoStatus.SetResult(string(complianceScan.Status.Result))
		protoStatus.SetErrorMessage(complianceScan.Status.ErrorMessage)
		protoStatus.SetCurrentIndex(complianceScan.Status.CurrentIndex)
		protoStatus.SetWarnings(complianceScan.Status.Warnings)
		protoStatus.SetRemainingRetries(int64(complianceScan.Status.RemainingRetries))
		protoStatus.SetStartTime(creationTime)

		if complianceScan.Status.EndTimestamp != nil {
			endTime, err := protocompat.ConvertTimeToTimestampOrError(complianceScan.Status.EndTimestamp.Time)
			if err != nil {
				log.Warnf("unable to convert end time %v", err)
			} else {
				protoStatus.SetEndTime(endTime)
			}
		}

		if complianceScan.Status.StartTimestamp != nil {
			startTime, err := protocompat.ConvertTimeToTimestampOrError(complianceScan.Status.StartTimestamp.Time)
			if err != nil {
				log.Warnf("unable to convert start time %v", err)
			} else {
				protoStatus.SetLastStartTime(startTime)
			}
		}

		protoScan := &central.ComplianceOperatorScanV2{}
		protoScan.SetId(uid)
		protoScan.SetName(complianceScan.Name)
		protoScan.SetProfileId(complianceScan.Spec.Profile)
		protoScan.SetLabels(complianceScan.Labels)
		protoScan.SetAnnotations(complianceScan.Annotations)
		protoScan.SetScanType(string(complianceScan.Spec.ScanType))
		protoScan.SetStatus(protoStatus)
		se2 := &central.SensorEvent{}
		se2.SetId(protoScan.GetId())
		se2.SetAction(action)
		se2.SetComplianceOperatorScanV2(proto.ValueOrDefault(protoScan))
		events = append(events, se2)
	}

	return component.NewEvent(events...)
}
