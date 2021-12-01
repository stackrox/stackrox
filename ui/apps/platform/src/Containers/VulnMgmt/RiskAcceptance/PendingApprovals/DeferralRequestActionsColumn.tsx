import React, { ReactElement } from 'react';
import { ActionsColumn } from '@patternfly/react-table';
import { VulnerabilityRequest } from './pendingApprovals.graphql';
import { RequestsToBeAssessed } from './types';

export type DeferralRequestActionsColumnProps = {
    row: VulnerabilityRequest;
    setRequestsToBeAssessed: React.Dispatch<React.SetStateAction<RequestsToBeAssessed>>;
};

function DeferralRequestActionsColumn({
    row,
    setRequestsToBeAssessed,
}: DeferralRequestActionsColumnProps): ReactElement {
    const items = [
        {
            title: 'Approve deferral',
            onClick: (event) => {
                event.preventDefault();
                setRequestsToBeAssessed({ type: 'APPROVE_DEFERRAL', requests: [row] });
            },
        },
        {
            title: 'Deny deferral',
            onClick: (event) => {
                event.preventDefault();
                setRequestsToBeAssessed({ type: 'DENY_DEFERRAL', requests: [row] });
            },
        },
    ];
    return <ActionsColumn items={items} />;
}

export default DeferralRequestActionsColumn;
