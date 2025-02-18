import React, { ReactElement } from 'react';
import { generatePath, useHistory } from 'react-router-dom';
import { ActionsColumn } from '@patternfly/react-table';

import { ComplianceScanConfigurationStatus } from 'services/ComplianceScanConfigurationService';

import { scanConfigDetailsPath } from './compliance.scanConfigs.routes';

// Component for scan configs table table page corresponds to ScanConfigActionDropdown for scan config details page.

// Caller is responsible for conditional rendering only if READ_WRITE_ACCESS level for Compliance resource.

export type ScanConfigActionsColumnProps = {
    handleDeleteScanConfig: (scanConfigResponse: ComplianceScanConfigurationStatus) => void;
    handleRunScanConfig: (scanConfigResponse: ComplianceScanConfigurationStatus) => void;
    handleSendReport: (scanConfigResponse: ComplianceScanConfigurationStatus) => void;
    handleGenerateDownload: (scanConfigResponse: ComplianceScanConfigurationStatus) => void;
    scanConfigResponse: ComplianceScanConfigurationStatus;
    isSnapshotStatusPending: boolean;
    isReportJobsEnabled: boolean;
};

function ScanConfigActionsColumn({
    handleDeleteScanConfig,
    handleRunScanConfig,
    handleSendReport,
    handleGenerateDownload,
    scanConfigResponse,
    isSnapshotStatusPending,
    isReportJobsEnabled,
}: ScanConfigActionsColumnProps): ReactElement {
    const history = useHistory();

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
            isDisabled: isSnapshotStatusPending,
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
            isDisabled: isSnapshotStatusPending,
        },
        {
            title: 'Send report',
            description:
                notifiers.length === 0
                    ? 'Send is disabled if no delivery destinations'
                    : /* isScanning
                      ? 'Send is disabled while scan is running'
                      : */ '',
            onClick: (event) => {
                event.preventDefault();
                handleSendReport(scanConfigResponse);
            },
            isDisabled: notifiers.length === 0 || isSnapshotStatusPending,
        },
        {
            title: 'Generate download',
            onClick: (event) => {
                event.preventDefault();
                handleGenerateDownload(scanConfigResponse);
            },
            isHidden: !isReportJobsEnabled,
            isDisabled: isSnapshotStatusPending,
        },
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
            isDisabled: isSnapshotStatusPending,
        },
    ].filter(({ isHidden }) => !isHidden);

    return (
        <ActionsColumn
            // menuAppendTo={() => document.body}
            items={items}
        />
    );
}

export default ScanConfigActionsColumn;
