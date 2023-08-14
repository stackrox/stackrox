import React, { ReactElement } from 'react';
import {
    CheckCircleIcon,
    DownloadIcon,
    ExclamationCircleIcon,
    HelpIcon,
    InProgressIcon,
    PendingIcon,
} from '@patternfly/react-icons';
import { Flex, FlexItem, Tooltip } from '@patternfly/react-core';

import { ReportSnapshot } from 'services/ReportsService.types';

export type ReportJobStatusProps = {
    reportSnapshot: ReportSnapshot;
};

const genericMsg =
    'An issue was encountered. Please try again later. If the issue persists, please contact support';

function ReportJobStatus({ reportSnapshot }: ReportJobStatusProps): ReactElement {
    const { reportStatus, isDownloadAvailable } = reportSnapshot;

    let statusColorClass = '';
    let statusIcon: ReactElement;
    let statusText: ReactElement;

    if (
        reportStatus.runState === 'SUCCESS' &&
        reportStatus.reportNotificationMethod === 'DOWNLOAD' &&
        isDownloadAvailable
    ) {
        statusColorClass = 'pf-u-info-color-100';
        statusIcon = <DownloadIcon title="Report download was successfully prepared" />;
        statusText = <p>Ready for download</p>;
    } else if (
        reportStatus.runState === 'SUCCESS' &&
        reportStatus.reportNotificationMethod === 'DOWNLOAD' &&
        !isDownloadAvailable
    ) {
        statusColorClass = 'pf-u-disabled-color-100';
        statusIcon = <DownloadIcon title="Report download was deleted" />;
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
                                The download was deleted. Please generate a new download if needed.
                            </div>
                        }
                    >
                        <HelpIcon />
                    </Tooltip>
                </FlexItem>
            </Flex>
        );
    } else if (reportStatus.runState === 'SUCCESS') {
        statusColorClass = 'pf-u-success-color-100';
        statusIcon = <CheckCircleIcon title="Report run was successful" />;
        statusText = <p className="pf-u-success-color-100">Successful</p>;
    } else if (reportStatus.runState === 'FAILURE') {
        statusColorClass = 'pf-u-danger-color-100';
        statusIcon = (
            <Tooltip content={reportStatus?.errorMsg || genericMsg}>
                <ExclamationCircleIcon title="Report run was unsuccessful" />
            </Tooltip>
        );
        statusText = <p>Error</p>;
    } else if (reportStatus.runState === 'PREPARING') {
        statusIcon = <InProgressIcon title="Report run is preparing" />;
        statusText = <p>Preparing</p>;
    } else {
        statusIcon = <PendingIcon title="Report run is waiting" />;
        statusText = <p>Waiting</p>;
    }

    return (
        <Flex alignItems={{ default: 'alignItemsCenter' }} className={statusColorClass}>
            <FlexItem>{statusIcon}</FlexItem>
            <FlexItem>{statusText}</FlexItem>
        </Flex>
    );
}

export default ReportJobStatus;
