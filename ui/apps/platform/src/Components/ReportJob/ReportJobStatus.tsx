import type { ReactElement } from 'react';
import {
    CheckCircleIcon,
    DownloadIcon,
    ExclamationCircleIcon,
    ExclamationTriangleIcon,
    HelpIcon,
    InProgressIcon,
    PendingIcon,
} from '@patternfly/react-icons';
import { Button, Flex, FlexItem, Tooltip } from '@patternfly/react-core';
import capitalize from 'lodash/capitalize';

import type { ReportStatus } from 'types/reportJob';
import PartialReportModal from './PartialReportModal';

export type ReportJobStatusProps = {
    reportStatus: ReportStatus;
    isDownloadAvailable: boolean;
    areDownloadActionsDisabled: boolean;
    onDownload: () => void;
};

const genericMsg =
    'An issue was encountered. Please try again later. If the issue persists, please contact support.';

function ReportJobStatus({
    reportStatus,
    isDownloadAvailable,
    areDownloadActionsDisabled,
    onDownload,
}: ReportJobStatusProps): ReactElement {
    const isDownload = reportStatus.reportNotificationMethod === 'DOWNLOAD';

    let statusIconColorClass = '';
    let statusTextColorClass = '';
    let statusIcon: ReactElement;
    let statusText: ReactElement;

    if (reportStatus.runState === 'PREPARING') {
        statusIcon = <InProgressIcon title="Report run is preparing" />;
        statusText = <p>Preparing</p>;
    } else if (reportStatus.runState === 'WAITING') {
        statusIcon = <PendingIcon title="Report run is waiting" />;
        statusText = <p>Waiting</p>;
    } else if (reportStatus.runState === 'FAILURE') {
        statusIconColorClass = 'pf-v6-u-icon-color-status-danger';
        statusTextColorClass = 'pf-v6-u-text-color-status-danger';
        statusIcon = (
            <Tooltip
                content={reportStatus?.errorMsg ? capitalize(reportStatus.errorMsg) : genericMsg}
            >
                <ExclamationCircleIcon title="Report run was unsuccessful" />
            </Tooltip>
        );
        statusText = <p>Report failed to generate</p>;
    } else if (isDownload && !isDownloadAvailable) {
        statusIconColorClass = 'pf-v6-u-icon-color-disabled';
        statusTextColorClass = 'pf-v6-u-text-color-disabled';
        statusIcon = <DownloadIcon title="Report download was deleted" />;
        statusText = (
            <Flex
                direction={{ default: 'row' }}
                spaceItems={{ default: 'spaceItemsSm' }}
                alignItems={{ default: 'alignItemsCenter' }}
            >
                <FlexItem>
                    <p>Report download deleted</p>
                </FlexItem>
                <FlexItem>
                    <Tooltip
                        content={
                            <div>
                                The download was deleted. Please generate a new download if needed.
                            </div>
                        }
                    >
                        <HelpIcon title="Download deletion explanation" />
                    </Tooltip>
                </FlexItem>
            </Flex>
        );
    } else if (isDownload && isDownloadAvailable && areDownloadActionsDisabled) {
        statusIconColorClass = 'pf-v6-u-icon-color-disabled';
        statusTextColorClass = 'pf-v6-u-text-color-disabled';
        statusIcon = <DownloadIcon title="Report download was successfully prepared" />;
        statusText = (
            <Flex
                direction={{ default: 'row' }}
                spaceItems={{ default: 'spaceItemsSm' }}
                alignItems={{ default: 'alignItemsCenter' }}
            >
                <FlexItem>
                    <p>Report ready for download</p>
                </FlexItem>
                <FlexItem>
                    <Tooltip
                        content={
                            <div>
                                Only the requestor of the download has the authority to access or
                                remove it.
                            </div>
                        }
                    >
                        <HelpIcon title="Permission limitations on download" />
                    </Tooltip>
                </FlexItem>
            </Flex>
        );
    } else if (
        isDownload &&
        isDownloadAvailable &&
        !areDownloadActionsDisabled &&
        reportStatus.runState === 'PARTIAL_SCAN_ERROR_DOWNLOAD'
    ) {
        statusIconColorClass = 'pf-v6-u-icon-color-brand';
        statusTextColorClass = 'pf-v6-u-text-color-brand';
        statusIcon = <DownloadIcon title="Partial report download was successfully prepared" />;
        statusText = (
            <PartialReportModal
                failedClusters={reportStatus.failedClusters}
                onDownload={onDownload}
            />
        );
    } else if (isDownload && isDownloadAvailable && !areDownloadActionsDisabled) {
        statusIconColorClass = 'pf-v6-u-icon-color-brand';
        statusTextColorClass = 'pf-v6-u-text-color-brand';
        statusIcon = <DownloadIcon title="Report download was successfully prepared" />;
        statusText = (
            <Button variant="link" isInline onClick={onDownload}>
                Report ready for download
            </Button>
        );
    } else if (reportStatus.runState === 'DELIVERED') {
        statusIconColorClass = 'pf-v6-u-icon-color-status-success';
        statusTextColorClass = 'pf-v6-u-text-color-status-success';
        statusIcon = <CheckCircleIcon title="Report was successfully sent" />;
        statusText = <p className="pf-v6-u-text-color-status-success">Report successfully sent</p>;
    } else if (reportStatus.runState === 'PARTIAL_SCAN_ERROR_EMAIL') {
        statusIconColorClass = 'pf-v6-u-icon-color-status-success';
        statusTextColorClass = 'pf-v6-u-text-color-status-success';
        statusIcon = <CheckCircleIcon title="Partial report was successfully sent" />;
        statusText = <PartialReportModal failedClusters={reportStatus.failedClusters} />;
    } else {
        statusIconColorClass = 'pf-v6-u-icon-color-status-warning';
        statusTextColorClass = 'pf-v6-u-text-color-status-warning';
        statusIcon = (
            <Tooltip content="Please contact support for more help.">
                <ExclamationTriangleIcon title="Report run status is unknown" />
            </Tooltip>
        );
        statusText = <p>Unknown status</p>;
    }

    return (
        <Flex alignItems={{ default: 'alignItemsCenter' }}>
            <FlexItem className={statusIconColorClass}>{statusIcon}</FlexItem>
            <FlexItem className={statusTextColorClass}>{statusText}</FlexItem>
        </Flex>
    );
}

export default ReportJobStatus;
