//go:build sql_integration

package tests

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// InjectProfile injects a ComplianceOperatorProfileV2 sensor event.
// productType should match the scan's ScanType (e.g. "Platform", "Node")
// for the ProfileRefId to link correctly.
func InjectProfile(s *FakeSensor, id, name, profileID, version, title, productType string) {
	s.Inject(NewSensorEvent(&central.SensorEvent{
		Id:     id,
		Action: central.ResourceAction_CREATE_RESOURCE,
		Resource: &central.SensorEvent_ComplianceOperatorProfileV2{
			ComplianceOperatorProfileV2: &central.ComplianceOperatorProfileV2{
				Id:             id,
				ProfileId:      profileID,
				Name:           name,
				ProfileVersion: version,
				Title:          title,
				Annotations: map[string]string{
					"compliance.openshift.io/product-type": productType,
				},
			},
		},
	}))
}

// InjectRule injects a ComplianceOperatorRuleV2 sensor event.
// The compliance.openshift.io/rule annotation is set automatically from name.
func InjectRule(s *FakeSensor, id, name, ruleType string, labels map[string]string) {
	s.Inject(NewSensorEvent(&central.SensorEvent{
		Id:     id,
		Action: central.ResourceAction_CREATE_RESOURCE,
		Resource: &central.SensorEvent_ComplianceOperatorRuleV2{
			ComplianceOperatorRuleV2: &central.ComplianceOperatorRuleV2{
				Id:       id,
				Name:     name,
				RuleType: ruleType,
				Labels:   labels,
				Annotations: map[string]string{
					"compliance.openshift.io/rule": name,
				},
			},
		},
	}))
}

// InjectSuite injects a ComplianceOperatorSuiteV2 sensor event.
func InjectSuite(s *FakeSensor, id, name string, status *central.ComplianceOperatorStatus) {
	s.Inject(NewSensorEvent(&central.SensorEvent{
		Id:     id,
		Action: central.ResourceAction_CREATE_RESOURCE,
		Resource: &central.SensorEvent_ComplianceOperatorSuiteV2{
			ComplianceOperatorSuiteV2: &central.ComplianceOperatorSuiteV2{
				Id:     id,
				Name:   name,
				Status: status,
			},
		},
	}))
}

// InjectSSB injects a ComplianceOperatorScanSettingBindingV2 sensor event.
func InjectSSB(s *FakeSensor, id, name string, profileNames []string, labels map[string]string) {
	s.Inject(NewSensorEvent(&central.SensorEvent{
		Id:     id,
		Action: central.ResourceAction_CREATE_RESOURCE,
		Resource: &central.SensorEvent_ComplianceOperatorScanSettingBindingV2{
			ComplianceOperatorScanSettingBindingV2: &central.ComplianceOperatorScanSettingBindingV2{
				Id:           id,
				Name:         name,
				ProfileNames: profileNames,
				Labels:       labels,
			},
		},
	}))
}

// InjectComplianceOperatorInfo sends a ComplianceOperatorInfo message
// indicating the operator is installed and healthy.
func InjectComplianceOperatorInfo(s *FakeSensor, version string) {
	s.Inject(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_ComplianceOperatorInfo{
			ComplianceOperatorInfo: &central.ComplianceOperatorInfo{
				Version:     version,
				Namespace:   "openshift-compliance",
				IsInstalled: true,
				TotalDesiredPodsOpt: &central.ComplianceOperatorInfo_TotalDesiredPods{
					TotalDesiredPods: 3,
				},
				TotalReadyPodsOpt: &central.ComplianceOperatorInfo_TotalReadyPods{
					TotalReadyPods: 3,
				},
			},
		},
	})
}

// InjectScan injects a ComplianceOperatorScanV2 sensor event.
// scanConfigName is the suite label value (compliance.openshift.io/suite).
// checkCount sets the check-count annotation; use -1 to omit it.
func InjectScan(s *FakeSensor, id, scanName, profileID, scanConfigName string, lastStartTime time.Time, checkCount int) {
	annotations := map[string]string{}
	if checkCount >= 0 {
		annotations["compliance.openshift.io/check-count"] = fmt.Sprintf("%d", checkCount)
	}
	s.Inject(NewSensorEvent(&central.SensorEvent{
		Id:     id,
		Action: central.ResourceAction_CREATE_RESOURCE,
		Resource: &central.SensorEvent_ComplianceOperatorScanV2{
			ComplianceOperatorScanV2: &central.ComplianceOperatorScanV2{
				Id:        id,
				Name:      scanName,
				ProfileId: profileID,
				Labels: map[string]string{
					"compliance.openshift.io/suite": scanConfigName,
				},
				Annotations: annotations,
				ScanType:    "Platform",
				Status: &central.ComplianceOperatorScanStatusV2{
					Phase:         "DONE",
					Result:        "CompliantAll",
					LastStartTime: timestamppb.New(lastStartTime),
				},
			},
		},
	}))
}

// InjectCheckResult injects a ComplianceOperatorCheckResultV2 sensor event.
// lastScannedTime is written to the compliance.openshift.io/last-scanned-timestamp annotation.
func InjectCheckResult(s *FakeSensor, id, checkID, checkName, scanName, suiteName, ruleName string, status central.ComplianceOperatorCheckResultV2_CheckStatus, lastScannedTime time.Time) {
	s.Inject(NewSensorEvent(&central.SensorEvent{
		Id:     id,
		Action: central.ResourceAction_CREATE_RESOURCE,
		Resource: &central.SensorEvent_ComplianceOperatorResultV2{
			ComplianceOperatorResultV2: &central.ComplianceOperatorCheckResultV2{
				Id:        id,
				CheckId:   checkID,
				CheckName: checkName,
				Status:    status,
				Severity:  central.ComplianceOperatorRuleSeverity_MEDIUM_RULE_SEVERITY,
				ScanName:  scanName,
				SuiteName: suiteName,
				Annotations: map[string]string{
					"compliance.openshift.io/last-scanned-timestamp": lastScannedTime.Format(time.RFC3339Nano),
					"compliance.openshift.io/rule":                   ruleName,
				},
			},
		},
	}))
}
