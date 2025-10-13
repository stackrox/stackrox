package compliancemanager

import "github.com/stackrox/rox/generated/internalapi/central"

func buildScanConfigSensorMsg(msgID string, cron string, profiles []string, configName string, createConfig bool) *central.MsgToSensor {
	if createConfig {
		return &central.MsgToSensor{
			Msg: &central.MsgToSensor_ComplianceRequest{
				ComplianceRequest: &central.ComplianceRequest{
					Request: &central.ComplianceRequest_ApplyScanConfig{
						ApplyScanConfig: &central.ApplyComplianceScanConfigRequest{
							Id: msgID,
							ScanRequest: &central.ApplyComplianceScanConfigRequest_ScheduledScan_{
								ScheduledScan: &central.ApplyComplianceScanConfigRequest_ScheduledScan{
									ScanSettings: &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
										ScanName:       configName,
										StrictNodeScan: true,
										Profiles:       profiles,
									},
									Cron: cron,
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
								ScanSettings: &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
									ScanName:       configName,
									StrictNodeScan: true,
									Profiles:       profiles,
								},
								Cron: cron,
							},
						},
					},
				},
			},
		},
	}
}
