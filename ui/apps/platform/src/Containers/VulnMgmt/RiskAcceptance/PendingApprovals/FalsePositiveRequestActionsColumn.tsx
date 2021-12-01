import React, { ReactElement } from 'react';
import { ActionsColumn } from '@patternfly/react-table';
import { VulnerabilityRequest } from './pendingApprovals.graphql';
import { RequestsToBeAssessed } from './types';

export type DeferralRequestActionsColumnProps = {
    row: VulnerabilityRequest;
    setRequestsToBeAssessed: React.Dispatch<React.SetStateAction<RequestsToBeAssessed>>;
};

function FalsePositiveRequestActionsColumn({ row, setRequestsToBeAssessed }): ReactElement {
    const items = [
        {
            title: 'Approve false positive',
            onClick: (event) => {
                event.preventDefault();
                setRequestsToBeAssessed({ type: 'APPROVE_FALSE_POSITIVE', requests: [row] });
            },
        },
        {
            title: 'Deny false positive',
            onClick: (event) => {
                event.preventDefault();
                setRequestsToBeAssessed({ type: 'DENY_FALSE_POSITIVE', requests: [row] });
            },
        },
    ];
    return <ActionsColumn items={items} />;
}

export default FalsePositiveRequestActionsColumn;
