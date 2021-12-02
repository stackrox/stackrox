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
                setRequestsToBeAssessed({
                    type: 'FALSE_POSITIVE',
                    action: 'APPROVE',
                    requests: [row],
                });
            },
        },
        {
            title: 'Deny false positive',
            onClick: (event) => {
                event.preventDefault();
                setRequestsToBeAssessed({
                    type: 'FALSE_POSITIVE',
                    action: 'DENY',
                    requests: [row],
                });
            },
        },
        {
            title: 'Cancel false positive',
            onClick: (event) => {
                event.preventDefault();
                setRequestsToBeAssessed({
                    type: 'FALSE_POSITIVE',
                    action: 'CANCEL',
                    requests: [row],
                });
            },
        },
    ];
    return <ActionsColumn items={items} />;
}

export default FalsePositiveRequestActionsColumn;
