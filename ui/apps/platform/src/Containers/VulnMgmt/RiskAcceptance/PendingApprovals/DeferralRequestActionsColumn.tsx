import React, { ReactElement } from 'react';
import { ActionsColumn, IActions } from '@patternfly/react-table';
import { VulnerabilityRequest } from '../vulnerabilityRequests.graphql';
import { RequestsToBeAssessed } from './types';

export type DeferralRequestActionsColumnProps = {
    row: VulnerabilityRequest;
    setRequestsToBeAssessed: React.Dispatch<React.SetStateAction<RequestsToBeAssessed>>;
    canApproveRequest: boolean;
    canCancelRequest: boolean;
};

function DeferralRequestActionsColumn({
    row,
    setRequestsToBeAssessed,
    canApproveRequest,
    canCancelRequest,
}: DeferralRequestActionsColumnProps): ReactElement {
    const items: IActions = [
        {
            title: 'Approve deferral',
            onClick: (event) => {
                event.preventDefault();
                setRequestsToBeAssessed({
                    type: 'DEFERRAL',
                    action: 'APPROVE',
                    requests: [row],
                });
            },
            isDisabled: !canApproveRequest,
        },
        {
            title: 'Deny deferral',
            onClick: (event) => {
                event.preventDefault();
                setRequestsToBeAssessed({ type: 'DEFERRAL', action: 'DENY', requests: [row] });
            },
            isDisabled: !canApproveRequest,
        },
        {
            title: 'Cancel deferral',
            onClick: (event) => {
                event.preventDefault();
                setRequestsToBeAssessed({
                    type: 'DEFERRAL',
                    action: 'CANCEL',
                    requests: [row],
                });
            },
            isDisabled: !canCancelRequest,
        },
    ];
    return <ActionsColumn items={items} />;
}

export default DeferralRequestActionsColumn;
