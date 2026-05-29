import type { ReactElement } from 'react';
import { Divider, DropdownItem } from '@patternfly/react-core';

import type { ComplianceScanConfigurationStatus } from 'services/ComplianceScanConfigurationService';
import MenuDropdown from 'Components/PatternFly/MenuDropdown';

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
    const { scanConfig } = scanConfigResponse;
    const { notifiers } = scanConfig;
    const isProcessing = isScanning || isReportStatusPending;

    return (
        <MenuDropdown
            toggleText="Actions"
            popperProps={{
                position: 'end',
            }}
        >
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
            <Divider component="li" key="separator" />
            <DropdownItem
                key="Send report"
                description={
                    notifiers.length === 0 ? 'Send is disabled if no delivery destinations' : ''
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
