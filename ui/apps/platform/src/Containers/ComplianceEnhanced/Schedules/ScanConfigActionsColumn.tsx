import type { ReactElement } from 'react';
import { generatePath, useNavigate } from 'react-router-dom-v5-compat';
import { ActionsColumn } from '@patternfly/react-table';

import type { ComplianceScanConfigurationStatus } from 'services/ComplianceScanConfigurationService';

import { scanConfigDetailsPath } from './compliance.scanConfigs.routes';

export type ScanConfigActionsColumnProps = {
    handleDeleteScanConfig: (scanConfigResponse: ComplianceScanConfigurationStatus) => void;
    handleRunScanConfig: (scanConfigResponse: ComplianceScanConfigurationStatus) => void;
    handleSendReport: (scanConfigResponse: ComplianceScanConfigurationStatus) => void;
    handleGenerateDownload: (scanConfigResponse: ComplianceScanConfigurationStatus) => void;
    scanConfigResponse: ComplianceScanConfigurationStatus;
    isSnapshotStatusPending: boolean;
};

function ScanConfigActionsColumn({
    handleDeleteScanConfig,
    handleRunScanConfig,
    handleSendReport,
    handleGenerateDownload,
    scanConfigResponse,
    isSnapshotStatusPending,
}: ScanConfigActionsColumnProps): ReactElement {
    const navigate = useNavigate();

    const { id, scanConfig, modifiedBy } = scanConfigResponse;
    const { notifiers } = scanConfig;
    const isDiscovered = !modifiedBy?.id;
    const scanConfigUrl = generatePath(scanConfigDetailsPath, {
        scanConfigId: id,
    });

    const items = [
        ...(isDiscovered
            ? [
                  {
                      title: 'Edit notifications',
                      onClick: (event) => {
                          event.preventDefault();
                          navigate(`${scanConfigUrl}?action=edit`);
                      },
                      isDisabled: isSnapshotStatusPending,
                  },
              ]
            : [
                  {
                      title: 'Edit scan schedule',
                      onClick: (event) => {
                          event.preventDefault();
                          navigate(`${scanConfigUrl}?action=edit`);
                      },
                      isDisabled: isSnapshotStatusPending,
                  },
              ]),
        {
            isSeparator: true,
        },
        {
            title: 'Run scan',
            onClick: (event) => {
                event.preventDefault();
                handleRunScanConfig(scanConfigResponse);
            },
            isDisabled: isSnapshotStatusPending,
        },
        {
            title: 'Send report',
            description:
                notifiers.length === 0 ? 'Send is disabled if no delivery destinations' : '',
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
            isDisabled: isSnapshotStatusPending,
        },
        ...(!isDiscovered
            ? [
                  {
                      isSeparator: true,
                  },
                  {
                      title: (
                          <span className="pf-v6-u-text-color-status-danger">
                              Delete scan schedule
                          </span>
                      ),
                      onClick: (event) => {
                          event.preventDefault();
                          handleDeleteScanConfig(scanConfigResponse);
                      },
                      isDisabled: isSnapshotStatusPending,
                  },
              ]
            : []),
    ];

    return <ActionsColumn items={items} />;
}

export default ScanConfigActionsColumn;
