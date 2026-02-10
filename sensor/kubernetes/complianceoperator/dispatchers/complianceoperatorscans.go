package dispatchers

import (
	"strings"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

// ScanDispatcher handles compliance operator scan objects
type ScanDispatcher struct {
	tailoredProfileLister cache.GenericLister
}

// NewScanDispatcher creates and returns a new scan dispatcher
func NewScanDispatcher(tailoredProfileLister cache.GenericLister) *ScanDispatcher {
	return &ScanDispatcher{
		tailoredProfileLister: tailoredProfileLister,
	}
}

// getProfileIDForScan returns the profile ID to use for the scan.
// For CEL scans (CustomRules), the Compliance Operator sets spec.profile to the TailoredProfile NAME,
// not the XCCDF ID. We need to look up the TailoredProfile and use its status.id for consistency
// with how profiles are stored in Central.
//
// Detection: Regular profiles and OpenSCAP scans use XCCDF IDs which start with "xccdf_".
// CEL scans reference TailoredProfiles by NAME (without the xccdf_ prefix).
func (c *ScanDispatcher) getProfileIDForScan(complianceScan *v1alpha1.ComplianceScan) string {
	profileID := complianceScan.Spec.Profile

	// If profile already has XCCDF format, no lookup needed (OpenSCAP scans)
	if strings.HasPrefix(profileID, "xccdf_") {
		return profileID
	}

	// If no lister is available, fall back to the original behavior
	if c.tailoredProfileLister == nil {
		return profileID
	}

	// Look up the TailoredProfile to get its status.id (XCCDF ID)
	tpObj, err := c.tailoredProfileLister.ByNamespace(complianceScan.GetNamespace()).Get(complianceScan.Spec.Profile)
	if err != nil {
		log.Debugf("Could not find TailoredProfile %s for scan %s: %v", complianceScan.Spec.Profile, complianceScan.Name, err)
		return profileID
	}

	tpUnstructured, ok := tpObj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("Fetched TailoredProfile not of type 'unstructured': %T", tpObj)
		return profileID
	}

	var tailoredProfile v1alpha1.TailoredProfile
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(tpUnstructured.Object, &tailoredProfile); err != nil {
		log.Errorf("Error converting unstructured to TailoredProfile: %v", err)
		return profileID
	}

	// Use the XCCDF ID from the TailoredProfile status
	if tailoredProfile.Status.ID != "" {
		log.Debugf("CEL scan %s: resolved ProfileId from %q to %q", complianceScan.Name, complianceScan.Spec.Profile, tailoredProfile.Status.ID)
		return tailoredProfile.Status.ID
	}

	return profileID
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

	// Get the correct profile ID - for CEL scans, we need to look up the XCCDF ID from the TailoredProfile
	profileID := c.getProfileIDForScan(&complianceScan)

	// We probably could have gotten away with re-using the storage proto here for the time being.
	// But we have a new field coming on for profiles and using the storage object even in an internal api
	// is a bad practice, so we will make that split now.  V1 and V2 compliance will both need to work for a period
	// of time.  However, we should not need to send the same profile twice, the pipeline can convert the V2 sensor message
	// so V1 and V2 objects can both be stored.

	protoScan := &storage.ComplianceOperatorScan{
		Id:          uid,
		Name:        complianceScan.Name,
		ProfileId:   profileID,
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
			ProfileId:   profileID,
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
