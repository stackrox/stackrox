import type { ReactElement } from 'react';
import { ActionsColumn } from '@patternfly/react-table';

import type { ComplianceScanConfigurationStatus } from 'services/ComplianceScanConfigurationService';

export type ScanConfigActionsColumnProps = {
    handleRunScanConfig: (scanConfigResponse: ComplianceScanConfigurationStatus) => void;
    handleSendReport: (scanConfigResponse: ComplianceScanConfigurationStatus) => void;
    handleGenerateDownload: (scanConfigResponse: ComplianceScanConfigurationStatus) => void;
    scanConfigResponse: ComplianceScanConfigurationStatus;
    isSnapshotStatusPending: boolean;
};

function ScanConfigActionsColumn({
    handleRunScanConfig,
    handleSendReport,
    handleGenerateDownload,
    scanConfigResponse,
    isSnapshotStatusPending,
}: ScanConfigActionsColumnProps): ReactElement {
    const { scanConfig } = scanConfigResponse;
    const { notifiers } = scanConfig;

    const { isManaged } = scanConfigResponse;
    const reportUnavailable = !isManaged;

    const items = [
        {
            title: 'Run scan',
            onClick: (event) => {
                event.preventDefault();
                handleRunScanConfig(scanConfigResponse);
            },
            isDisabled: isSnapshotStatusPending,
        },
        {
            isSeparator: true,
        },
        {
            title: 'Send report',
            description: reportUnavailable
                ? 'Reports are not available for externally managed configurations'
                : notifiers.length === 0
                  ? 'Send is disabled if no delivery destinations'
                  : '',
            onClick: (event) => {
                event.preventDefault();
                handleSendReport(scanConfigResponse);
            },
            isDisabled: reportUnavailable || notifiers.length === 0 || isSnapshotStatusPending,
        },
        {
            title: 'Generate download',
            description: reportUnavailable
                ? 'Reports are not available for externally managed configurations'
                : '',
            onClick: (event) => {
                event.preventDefault();
                handleGenerateDownload(scanConfigResponse);
            },
            isDisabled: reportUnavailable || isSnapshotStatusPending,
        },
    ];

    return <ActionsColumn items={items} />;
}

export default ScanConfigActionsColumn;
