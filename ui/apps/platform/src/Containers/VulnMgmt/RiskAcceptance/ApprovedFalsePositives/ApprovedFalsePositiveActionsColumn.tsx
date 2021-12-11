import React, { ReactElement } from 'react';
import { ActionsColumn } from '@patternfly/react-table';
import { VulnerabilityRequest } from '../vulnerabilityRequests.graphql';
import { ApprovedFalsePositiveRequestsToBeAssessed } from './types';

export type ApprovedFalsePositiveActionsColumnProps = {
    row: VulnerabilityRequest;
    setRequestsToBeAssessed: React.Dispatch<
        React.SetStateAction<ApprovedFalsePositiveRequestsToBeAssessed>
    >;
};

function ApprovedFalsePositiveActionsColumn({
    row,
    setRequestsToBeAssessed,
}: ApprovedFalsePositiveActionsColumnProps): ReactElement {
    const items = [
        {
            title: 'Reobserve CVE',
            onClick: (event) => {
                event.preventDefault();
                setRequestsToBeAssessed({
                    type: 'FALSE_POSITIVE',
                    action: 'UNDO',
                    requests: [row],
                });
            },
        },
    ];
    return <ActionsColumn items={items} />;
}

export default ApprovedFalsePositiveActionsColumn;
