import React, { ReactElement, useState } from 'react';
import { generatePath, useHistory } from 'react-router-dom';
import {
    Dropdown,
    DropdownItem,
    DropdownSeparator,
    DropdownToggle,
} from '@patternfly/react-core/deprecated';
import { CaretDownIcon } from '@patternfly/react-icons';

import { ComplianceScanConfigurationStatus } from 'services/ComplianceScanConfigurationService';

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
    isReportJobsEnabled: boolean;
};

function ScanConfigActionDropdown({
    handleRunScanConfig,
    handleSendReport,
    handleGenerateDownload,
    isScanning,
    isReportStatusPending,
    scanConfigResponse,
    isReportJobsEnabled,
}: ScanConfigActionDropdownProps): ReactElement {
    const history = useHistory();

    const [isOpen, setIsOpen] = useState(false);

    const { id, scanConfig } = scanConfigResponse;
    const { notifiers } = scanConfig;
    const scanConfigUrl = generatePath(scanConfigDetailsPath, {
        scanConfigId: id,
    });
    const isProcessing = isScanning || isReportStatusPending;

    function onToggle() {
        setIsOpen((prevValue) => !prevValue);
    }

    function onSelect() {
        setIsOpen(false);
    }

    const dropdownItems = [
        <DropdownItem
            key="Edit scan schedule"
            component="button"
            // description={isScanning ? 'Edit is disabled while scan is running' : ''}
            isDisabled={isProcessing}
            onClick={() => {
                history.push({
                    pathname: scanConfigUrl,
                    search: 'action=edit',
                });
            }}
        >
            Edit scan schedule
        </DropdownItem>,
        <DropdownSeparator key="Separator" />,
        <DropdownItem
            key="Run scan"
            component="button"
            description={isScanning ? 'Run is disabled while scan is already running' : ''}
            isDisabled={isProcessing}
            onClick={() => {
                handleRunScanConfig(scanConfigResponse);
            }}
        >
            Run scan
        </DropdownItem>,
        <DropdownItem
            key="Send report"
            component="button"
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
        </DropdownItem>,
    ];

    if (isReportJobsEnabled) {
        dropdownItems.push(
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
        );
    }

    return (
        <Dropdown
            onSelect={onSelect}
            position="right"
            toggle={
                <DropdownToggle onToggle={onToggle} toggleIndicator={CaretDownIcon}>
                    Actions
                </DropdownToggle>
            }
            isOpen={isOpen}
            dropdownItems={dropdownItems}
        />
    );
}

export default ScanConfigActionDropdown;
