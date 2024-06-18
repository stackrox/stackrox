import React, { ReactElement, useState } from 'react';
import { generatePath, useHistory } from 'react-router-dom';
import {
    Dropdown,
    DropdownItem,
    DropdownSeparator,
    DropdownToggle,
} from '@patternfly/react-core/deprecated';
import { CaretDownIcon } from '@patternfly/react-icons';

import useFeatureFlags from 'hooks/useFeatureFlags';
import { ComplianceScanConfigurationStatus } from 'services/ComplianceScanConfigurationService';

import { scanConfigDetailsPath } from './compliance.scanConfigs.routes';

// Component for scan config details page corresponds to ScanConfigActionsColumn for scan configs table table page.
// One difference: omit delete on details page.

// Caller is responsible for conditional rendering only if READ_WRITE_ACCESS level for Compliance resource.

export type ScanConfigActionDropdownProps = {
    handleRunScanConfig: (scanConfigResponse: ComplianceScanConfigurationStatus) => void;
    handleSendReport: (scanConfigResponse: ComplianceScanConfigurationStatus) => void;
    isScanning: boolean;
    scanConfigResponse: ComplianceScanConfigurationStatus;
};

function ScanConfigActionDropdown({
    handleRunScanConfig,
    handleSendReport,
    isScanning,
    scanConfigResponse,
}: ScanConfigActionDropdownProps): ReactElement {
    const history = useHistory();
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isComplianceReportingEnabled = isFeatureFlagEnabled('ROX_COMPLIANCE_REPORTING');

    const [isOpen, setIsOpen] = useState(false);

    const { id, scanConfig } = scanConfigResponse;
    const { notifiers } = scanConfig;
    const scanConfigUrl = generatePath(scanConfigDetailsPath, {
        scanConfigId: id,
    });

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
            // isDisabled={isScanning}
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
            key="Run scan now"
            component="button"
            description={isScanning ? 'Run is disabled while scan is already running' : ''}
            isDisabled={isScanning}
            onClick={() => {
                handleRunScanConfig(scanConfigResponse);
            }}
        >
            Run scan now
        </DropdownItem>,
    ];

    if (isComplianceReportingEnabled) {
        /* eslint-disable no-nested-ternary */
        dropdownItems.push(
            <DropdownItem
                key="Send report now"
                component="button"
                description={
                    notifiers.length === 0
                        ? 'Send is disabled if no delivery destinations'
                        : /* : isScanning
                          ? 'Send is disabled while scan is running' */
                          ''
                }
                isDisabled={notifiers.length === 0 /* || isScanning */}
                onClick={() => {
                    handleSendReport(scanConfigResponse);
                }}
            >
                Send report now
            </DropdownItem>
        );
        /* eslint-enable no-nested-ternary */
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
