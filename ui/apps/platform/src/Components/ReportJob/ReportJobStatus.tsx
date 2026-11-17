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
import { Button, Flex, FlexItem, Icon, Tooltip } from '@patternfly/react-core';

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

    let statusIcon: ReactElement;
    let statusText: ReactElement;
    let contentOfTooltip = '';

    if (reportStatus.runState === 'PREPARING') {
        statusIcon = (
            <Icon>
                <InProgressIcon />
            </Icon>
        );
        statusText = <p>Preparing</p>;
    } else if (reportStatus.runState === 'WAITING') {
        statusIcon = (
            <Icon>
                <PendingIcon />
            </Icon>
        );
        statusText = <p>Waiting</p>;
    } else if (reportStatus.runState === 'FAILURE') {
        statusIcon = (
            <Icon status="danger">
                <ExclamationCircleIcon />
            </Icon>
        );
        statusText = <p>Report failed to generate</p>;
        contentOfTooltip = reportStatus?.errorMsg || genericMsg;
    } else if (isDownload && !isDownloadAvailable) {
        statusIcon = (
            <Icon>
                <DownloadIcon />
            </Icon>
        );
        statusText = <p>Report download deleted</p>;
        contentOfTooltip = 'The download was deleted. Please generate a new download if needed.';
    } else if (isDownload && isDownloadAvailable && areDownloadActionsDisabled) {
        statusIcon = (
            <Icon status="warning">
                <DownloadIcon />
            </Icon>
        );
        statusText = <p>Report ready for download</p>;
        contentOfTooltip =
            'Only the requestor of the download has the authority to access or remove it.';
    } else if (
        isDownload &&
        isDownloadAvailable &&
        !areDownloadActionsDisabled &&
        reportStatus.runState === 'PARTIAL_SCAN_ERROR_DOWNLOAD'
    ) {
        statusIcon = (
            <Icon status="warning">
                <DownloadIcon />
            </Icon>
        );
        statusText = (
            <PartialReportModal
                failedClusters={reportStatus.failedClusters}
                onDownload={onDownload}
            />
        );
    } else if (isDownload && isDownloadAvailable && !areDownloadActionsDisabled) {
        statusIcon = (
            <Icon status="success">
                <DownloadIcon />
            </Icon>
        );
        statusText = (
            <Button variant="link" isInline onClick={onDownload}>
                Report ready for download
            </Button>
        );
    } else if (reportStatus.runState === 'DELIVERED') {
        statusIcon = (
            <Icon status="success">
                <CheckCircleIcon />
            </Icon>
        );
        statusText = <p>Report successfully sent</p>;
    } else if (reportStatus.runState === 'PARTIAL_SCAN_ERROR_EMAIL') {
        statusIcon = (
            <Icon status="success">
                <CheckCircleIcon />
            </Icon>
        );
        statusText = <PartialReportModal failedClusters={reportStatus.failedClusters} />;
    } else {
        statusIcon = (
            <Icon status="warning">
                <ExclamationTriangleIcon />
            </Icon>
        );
        statusText = <p>Unknown status</p>;
        contentOfTooltip = 'Please contact support for help.';
    }

    return (
        <Flex alignItems={{ default: 'alignItemsCenter' }} flexWrap={{ default: 'nowrap' }}>
            <FlexItem>{statusIcon}</FlexItem>
            <FlexItem>
                {contentOfTooltip ? (
                    <Flex
                        direction={{ default: 'row' }}
                        spaceItems={{ default: 'spaceItemsSm' }}
                        alignItems={{ default: 'alignItemsCenter' }}
                    >
                        <FlexItem>{statusText}</FlexItem>
                        <FlexItem>
                            <Tooltip content={contentOfTooltip} isContentLeftAligned>
                                <HelpIcon />
                            </Tooltip>
                        </FlexItem>
                    </Flex>
                ) : (
                    statusText
                )}
            </FlexItem>
        </Flex>
    );
}

export default ReportJobStatus;
