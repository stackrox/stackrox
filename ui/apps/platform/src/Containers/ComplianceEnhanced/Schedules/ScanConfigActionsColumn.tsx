import React, { ReactElement } from 'react';
import { generatePath, useHistory } from 'react-router-dom';
import { ActionsColumn } from '@patternfly/react-table';

import useFeatureFlags from 'hooks/useFeatureFlags';
import { ComplianceScanConfigurationStatus } from 'services/ComplianceScanConfigurationService';

import { scanConfigDetailsPath } from './compliance.scanConfigs.routes';

// Component for scan configs table table page corresponds to ScanConfigActionDropdown for scan config details page.

// Caller is responsible for conditional rendering only if READ_WRITE_ACCESS level for Compliance resource.

export type ScanConfigActionsColumnProps = {
    handleDeleteScanConfig: (scanConfigResponse: ComplianceScanConfigurationStatus) => void;
    handleRunScanConfig: (scanConfigResponse: ComplianceScanConfigurationStatus) => void;
    handleSendReport: (scanConfigResponse: ComplianceScanConfigurationStatus) => void;
    scanConfigResponse: ComplianceScanConfigurationStatus;
};

function ScanConfigActionsColumn({
    handleDeleteScanConfig,
    handleRunScanConfig,
    handleSendReport,
    scanConfigResponse,
}: ScanConfigActionsColumnProps): ReactElement {
    const history = useHistory();
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isComplianceReportingEnabled = isFeatureFlagEnabled('ROX_COMPLIANCE_REPORTING');

    const { id, /* lastExecutedTime, */ scanConfig } = scanConfigResponse;
    const { notifiers } = scanConfig;
    const scanConfigUrl = generatePath(scanConfigDetailsPath, {
        scanConfigId: id,
    });
    // const isScanning = lastExecutedTime === null;

    const items = [
        {
            title: 'Edit scan schedule',
            // description: isScanning ? 'Edit is disabled while scan is running' : '',
            // isDisabled: isScanning,
            onClick: (event) => {
                event.preventDefault();
                history.push({
                    pathname: scanConfigUrl,
                    search: 'action=edit',
                });
            },
        },
        {
            isSeparator: true,
        },
        {
            title: 'Run scan',
            // description: isScanning ? 'Run is disabled while scan is already running' : '',
            // isDisabled: isScanning,
            onClick: (event) => {
                event.preventDefault();
                handleRunScanConfig(scanConfigResponse);
            },
        },
        /* eslint-disable no-nested-ternary */
        {
            title: 'Send report',
            description:
                notifiers.length === 0
                    ? 'Send is disabled if no delivery destinations'
                    : /* isScanning
                      ? 'Send is disabled while scan is running'
                      : */ '',
            isDisabled: notifiers.length === 0 /* || isScanning */,
            onClick: (event) => {
                event.preventDefault();
                handleSendReport(scanConfigResponse);
            },
        },
        /* eslint-enable no-nested-ternary */
        {
            isSeparator: true,
        },
        {
            title: (
                <span className={/* isScanning ? '' : */ 'pf-v5-u-danger-color-100'}>
                    Delete scan schedule
                </span>
            ),
            // description: isScanning ? 'Delete is disabled while scan is running' : '',
            // isDisabled: isScanning,
            onClick: (event) => {
                event.preventDefault();
                handleDeleteScanConfig(scanConfigResponse);
            },
        },
    ].filter(({ title }) => title !== 'Send report' || isComplianceReportingEnabled);

    return (
        <ActionsColumn
            // menuAppendTo={() => document.body}
            items={items}
        />
    );
}

export default ScanConfigActionsColumn;
