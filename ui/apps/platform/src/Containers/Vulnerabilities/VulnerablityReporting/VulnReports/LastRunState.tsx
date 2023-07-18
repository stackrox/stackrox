import React, { ReactElement } from 'react';
import { Flex, FlexItem, Spinner } from '@patternfly/react-core';

import { getDateTime } from 'utils/dateUtils';
import { ReportStatus } from 'services/ReportsService.types';

type LastRunStateProps = {
    reportStatus: ReportStatus | null;
};

function LastRunState({ reportStatus }: LastRunStateProps): ReactElement {
    let statusIcon: ReactElement | null = null;
    let statusText = '';

    if (reportStatus?.runState === 'SUCCESS' || reportStatus?.runState === 'FAILURE') {
        statusText = getDateTime(reportStatus?.completedAt);
    } else if (reportStatus?.runState === 'PREPARING') {
        statusIcon = <Spinner isSVG size="sm" aria-label="Preparing report" aria-valuetext="" />;
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
