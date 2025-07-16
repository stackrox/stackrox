import React, { ReactElement } from 'react';
import { Divider, DropdownItem } from '@patternfly/react-core';
import { generatePath, useNavigate } from 'react-router-dom-v5-compat';

import { ComplianceScanConfigurationStatus } from 'services/ComplianceScanConfigurationService';
import MenuDropdown from 'Components/PatternFly/MenuDropdown';

import { scanConfigDetailsPath } from './compliance.scanConfigs.routes';

// Component for scan config details page corresponds to ScanConfigActionsColumn for scan configs table table page.
// One difference: omit delete on details page.

// Caller is responsible for conditional rendering only if READ_WRITE_ACCESS level for Compliance resource.

export type ScanConfigActionDropdownProps = {
    handleRunScanConfig: (scanConfigResponse: ComplianceScanConfigurationStatus) => void;
    handleSendReport: (scanConfigResponse: ComplianceScanConfigurationStatus) => void;
    handleGenerateDownload: (scanConfigResponse: ComplianceScanConfigurationStatus) => void;
    isScanning: boolean;
    isReportStatusPending: boolean;
    scanConfigResponse: ComplianceScanConfigurationStatus;
};

function ScanConfigActionDropdown({
    handleRunScanConfig,
    handleSendReport,
    handleGenerateDownload,
    isScanning,
    isReportStatusPending,
    scanConfigResponse,
}: ScanConfigActionDropdownProps): ReactElement {
    const navigate = useNavigate();

    const { id, scanConfig } = scanConfigResponse;
    const { notifiers } = scanConfig;
    const scanConfigUrl = generatePath(scanConfigDetailsPath, {
        scanConfigId: id,
    });
    const isProcessing = isScanning || isReportStatusPending;

    return (
        <MenuDropdown
            toggleText="Actions"
            popperProps={{
                position: 'end',
            }}
        >
            <DropdownItem
                key="Edit scan schedule"
                // description={isScanning ? 'Edit is disabled while scan is running' : ''}
                isDisabled={isProcessing}
                onClick={() => {
                    navigate(`${scanConfigUrl}?action=edit`);
                }}
            >
                Edit scan schedule
            </DropdownItem>
            <Divider component="li" key="separator" />
            <DropdownItem
                key="Run scan"
                description={isScanning ? 'Run is disabled while scan is already running' : ''}
                isDisabled={isProcessing}
                onClick={() => {
                    handleRunScanConfig(scanConfigResponse);
                }}
            >
                Run scan
            </DropdownItem>
            <DropdownItem
                key="Send report"
                description={
                    notifiers.length === 0
                        ? 'Send is disabled if no delivery destinations'
                        : /* : isScanning
                        ? 'Send is disabled while scan is running' */
                          ''
                }
                isDisabled={notifiers.length === 0 || isProcessing}
                onClick={() => {
                    handleSendReport(scanConfigResponse);
                }}
            >
                Send report
            </DropdownItem>
            <DropdownItem
                key="Generate download"
                component="button"
                isDisabled={isProcessing}
                onClick={() => {
                    handleGenerateDownload(scanConfigResponse);
                }}
            >
                Generate download
            </DropdownItem>
        </MenuDropdown>
    );
}

export default ScanConfigActionDropdown;
