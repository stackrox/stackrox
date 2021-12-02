import React, { ReactElement } from 'react';
import { ActionsColumn } from '@patternfly/react-table';
import { VulnerabilityRequest } from '../vulnerabilityRequests.graphql';
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
                setRequestsToBeAssessed({ type: 'DEFERRAL', action: 'APPROVE', requests: [row] });
            },
        },
        {
            title: 'Deny deferral',
            onClick: (event) => {
                event.preventDefault();
                setRequestsToBeAssessed({ type: 'DEFERRAL', action: 'DENY', requests: [row] });
            },
        },
        {
            title: 'Cancel deferral',
            onClick: (event) => {
                event.preventDefault();
                setRequestsToBeAssessed({ type: 'DEFERRAL', action: 'CANCEL', requests: [row] });
            },
        },
    ];
    return <ActionsColumn items={items} />;
}

export default DeferralRequestActionsColumn;
