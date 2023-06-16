import React, { ReactElement } from 'react';
import { ReportStatus } from 'types/report.proto';
import { Flex, FlexItem, Spinner } from '@patternfly/react-core';

import { getDateTime } from 'utils/dateUtils';

type LastRunStateProps = {
    reportStatus: ReportStatus;
};

function LastRunState({ reportStatus }: LastRunStateProps): ReactElement {
    let statusIcon: ReactElement | null = null;
    let statusText = '';

    if (reportStatus.runState === 'SUCCESS' || reportStatus.runState === 'FAILURE') {
        statusText = getDateTime(reportStatus.runTime);
    } else if (reportStatus.runState === 'PREPARING') {
        statusIcon = <Spinner isSVG size="sm" aria-label="Preparing report" />;
        statusText = 'Preparing';
    } else {
        statusText = 'Never run';
    }

    return (
        <Flex alignItems={{ default: 'alignItemsCenter' }}>
            {statusIcon && <FlexItem>{statusIcon}</FlexItem>}
            <FlexItem>{statusText}</FlexItem>
        </Flex>
    );
}

export default LastRunState;
