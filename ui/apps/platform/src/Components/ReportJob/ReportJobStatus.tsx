import React, { ReactElement } from 'react';
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

import { ReportStatus } from 'types/reportJob';
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

    let statusColorClass = '';
    let statusIcon: ReactElement;
    let statusText: ReactElement;

    if (reportStatus.runState === 'PREPARING') {
        statusIcon = <InProgressIcon title="Report run is preparing" />;
        statusText = <p>Preparing</p>;
    } else if (reportStatus.runState === 'WAITING') {
        statusIcon = <PendingIcon title="Report run is waiting" />;
        statusText = <p>Waiting</p>;
    } else if (reportStatus.runState === 'FAILURE') {
        statusColorClass = 'pf-v5-u-danger-color-100';
        statusIcon = (
            <Tooltip
                content={reportStatus?.errorMsg ? capitalize(reportStatus.errorMsg) : genericMsg}
            >
                <ExclamationCircleIcon title="Report run was unsuccessful" />
            </Tooltip>
        );
        statusText = <p>Error</p>;
    } else if (isDownload && isDownloadAvailable && areDownloadActionsDisabled) {
        statusColorClass = 'pf-v5-u-disabled-color-100';
        statusIcon = <DownloadIcon title="Report download was successfully prepared" />;
        statusText = (
            <Flex
                direction={{ default: 'row' }}
                spaceItems={{ default: 'spaceItemsSm' }}
                alignItems={{ default: 'alignItemsCenter' }}
            >
                <FlexItem>
                    <p>Ready for download</p>
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
        reportStatus.runState === 'PARTIAL_ERROR'
    ) {
        statusIcon = (
            <ExclamationTriangleIcon
                title="Report download prepared with partial errors"
                className="pf-v5-u-warning-color-100"
            />
        );
        statusText = (
            <PartialReportModal
                failedClusters={reportStatus.failedClusters}
                onConfirm={onDownload}
            />
        );
    } else if (!isDownload && reportStatus.runState === 'PARTIAL_ERROR') {
        statusIcon = (
            <ExclamationTriangleIcon
                title="Report was sent with partial errors"
                className="pf-v5-u-warning-color-100"
            />
        );
        statusText = (
            <PartialReportModal
                failedClusters={reportStatus.failedClusters}
            />
        );
        statusText =
            <p className="pf-v5-u-success-color-100">Successfully sent</p>;
    } else if (isDownload && isDownloadAvailable && !areDownloadActionsDisabled) {
        statusColorClass = 'pf-v5-u-primary-color-100';
        statusIcon = <DownloadIcon title="Report download was successfully prepared" />;
        statusText = (
            <Button variant="link" isInline className={statusColorClass} onClick={onDownload}>
                Ready for download
            </Button>
        );
    } else if (isDownload && !isDownloadAvailable) {
        statusColorClass = 'pf-v5-u-disabled-color-100';
        statusIcon = <DownloadIcon title="Report download was deleted" />;
        statusText = (
            <Flex
                direction={{ default: 'row' }}
                spaceItems={{ default: 'spaceItemsSm' }}
                alignItems={{ default: 'alignItemsCenter' }}
            >
                <FlexItem>
                    <p>Download deleted</p>
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
    } else if (reportStatus.runState === 'DELIVERED') {
        statusColorClass = 'pf-v5-u-success-color-100';
        statusIcon = <CheckCircleIcon title="Report was successfully sent"/>;
        statusText =
            <p className="pf-v5-u-success-color-100">Successfully sent</p>;
    } else {
        statusColorClass = 'pf-v5-u-warning-color-100';
        statusIcon = (
            <Tooltip content="Please contact support for more help.">
                <ExclamationTriangleIcon title="Report run status is unknown" />
            </Tooltip>
        );
        statusText = <p>Unknown status</p>;
    }

    return (
        <Flex alignItems={{ default: 'alignItemsCenter' }} className={statusColorClass}>
            <FlexItem>{statusIcon}</FlexItem>
            <FlexItem>{statusText}</FlexItem>
        </Flex>
    );
}

export default ReportJobStatus;
