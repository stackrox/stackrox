package compliancemanager

import "github.com/stackrox/rox/generated/internalapi/central"

func buildScanConfigSensorMsg(msgID string, cron string, profiles []string, profileRefs []*central.ApplyComplianceScanConfigRequest_BaseScanSettings_ProfileReference, configName string, createConfig bool, nodeRoles []string) *central.MsgToSensor {
	baseScanSettings := &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
		ScanName:       configName,
		StrictNodeScan: true,
		Profiles:       profiles,
		ProfileRefs:    profileRefs,
		NodeRoles:      nodeRoles,
	}

	if createConfig {
		return &central.MsgToSensor{
			Msg: &central.MsgToSensor_ComplianceRequest{
				ComplianceRequest: &central.ComplianceRequest{
					Request: &central.ComplianceRequest_ApplyScanConfig{
						ApplyScanConfig: &central.ApplyComplianceScanConfigRequest{
							Id: msgID,
							ScanRequest: &central.ApplyComplianceScanConfigRequest_ScheduledScan_{
								ScheduledScan: &central.ApplyComplianceScanConfigRequest_ScheduledScan{
									ScanSettings: baseScanSettings,
									Cron:         cron,
								},
							},
						},
					},
				},
			},
		}
	}

	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_ComplianceRequest{
			ComplianceRequest: &central.ComplianceRequest{
				Request: &central.ComplianceRequest_ApplyScanConfig{
					ApplyScanConfig: &central.ApplyComplianceScanConfigRequest{
						Id: msgID,
						ScanRequest: &central.ApplyComplianceScanConfigRequest_UpdateScan{
							UpdateScan: &central.ApplyComplianceScanConfigRequest_UpdateScheduledScan{
								ScanSettings: baseScanSettings,
								Cron:         cron,
							},
						},
					},
				},
			},
		},
	}
}
