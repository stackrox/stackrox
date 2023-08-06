import React, { ReactElement } from 'react';
import { Flex, FlexItem } from '@patternfly/react-core';

import { ReportStatus } from 'services/ReportsService.types';
import { InProgressIcon, PendingIcon } from '@patternfly/react-icons';

type MyActiveJobProps = {
    reportStatus: ReportStatus | undefined;
};

function MyActiveJobStatus({ reportStatus }: MyActiveJobProps): ReactElement {
    let statusIcon: ReactElement | null = null;
    let statusText;

    if (reportStatus?.runState === 'PREPARING') {
        statusIcon = <InProgressIcon title="Report run is preparing" />;
        statusText = 'Preparing';
    } else if (reportStatus?.runState === 'WAITING') {
        statusIcon = <PendingIcon title="Report run is waiting" />;
        statusText = 'Waiting';
    } else {
        statusText = '-';
    }

    return (
        <Flex alignItems={{ default: 'alignItemsCenter' }}>
            {statusIcon && <FlexItem>{statusIcon}</FlexItem>}
            <FlexItem>{statusText}</FlexItem>
        </Flex>
    );
}

export default MyActiveJobStatus;
