import React, { ReactElement } from 'react';
import { ReportStatus } from 'types/report.proto';
import { CheckCircleIcon, ExclamationCircleIcon } from '@patternfly/react-icons';
import { Flex, FlexItem, Tooltip } from '@patternfly/react-core';

type LastRunStatusStateProps = {
    reportStatus: ReportStatus;
};

const errorColor = 'var(--pf-global--danger-color--100)';
const successColor = 'var(--pf-global--success-color--100)';

const genericMsg =
    'An issue was encountered. Please try again later. If the issue persists, please contact support';

function LastRunStatusState({ reportStatus }: LastRunStatusStateProps): ReactElement {
    let statusIcon: ReactElement | null = null;
    let statusText = '-';

    if (reportStatus.runState === 'SUCCESS') {
        statusIcon = <CheckCircleIcon color={successColor} />;
    }
    if (reportStatus.runState === 'FAILURE') {
        statusIcon = (
            <Tooltip content={reportStatus.errorMsg || genericMsg}>
                <ExclamationCircleIcon color={errorColor} />
            </Tooltip>
        );
    }

    if (reportStatus.runState === 'SUCCESS' && reportStatus.reportNotificationMethod === 'EMAIL') {
        statusText = 'Emailed';
    } else if (
        reportStatus.runState === 'SUCCESS' &&
        reportStatus.reportNotificationMethod === 'DOWNLOAD'
    ) {
        statusText = 'Download prepared';
    } else if (
        reportStatus.runState === 'FAILURE' &&
        reportStatus.reportNotificationMethod === 'EMAIL'
    ) {
        statusText = 'Email attempted';
    } else if (
        reportStatus.runState === 'FAILURE' &&
        reportStatus.reportNotificationMethod === 'DOWNLOAD'
    ) {
        statusText = 'Failed to generate download';
    } else if (reportStatus.runState === 'SUCCESS') {
        statusText = 'Success';
    } else if (reportStatus.runState === 'FAILURE') {
        statusText = 'Error';
    }

    return (
        <Flex alignItems={{ default: 'alignItemsCenter' }}>
            {statusIcon && <FlexItem>{statusIcon}</FlexItem>}
            <FlexItem>{statusText}</FlexItem>
        </Flex>
    );
}

export default LastRunStatusState;
