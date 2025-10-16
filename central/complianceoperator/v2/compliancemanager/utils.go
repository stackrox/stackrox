package compliancemanager

import "github.com/stackrox/rox/generated/internalapi/central"

func buildScanConfigSensorMsg(msgID string, cron string, profiles []string, configName string, createConfig bool) *central.MsgToSensor {
	if createConfig {
		return central.MsgToSensor_builder{
			ComplianceRequest: central.ComplianceRequest_builder{
				ApplyScanConfig: central.ApplyComplianceScanConfigRequest_builder{
					Id: msgID,
					ScheduledScan: central.ApplyComplianceScanConfigRequest_ScheduledScan_builder{
						ScanSettings: central.ApplyComplianceScanConfigRequest_BaseScanSettings_builder{
							ScanName:       configName,
							StrictNodeScan: true,
							Profiles:       profiles,
						}.Build(),
						Cron: cron,
					}.Build(),
				}.Build(),
			}.Build(),
		}.Build()
	}

	return central.MsgToSensor_builder{
		ComplianceRequest: central.ComplianceRequest_builder{
			ApplyScanConfig: central.ApplyComplianceScanConfigRequest_builder{
				Id: msgID,
				UpdateScan: central.ApplyComplianceScanConfigRequest_UpdateScheduledScan_builder{
					ScanSettings: central.ApplyComplianceScanConfigRequest_BaseScanSettings_builder{
						ScanName:       configName,
						StrictNodeScan: true,
						Profiles:       profiles,
					}.Build(),
					Cron: cron,
				}.Build(),
			}.Build(),
		}.Build(),
	}.Build()
}
