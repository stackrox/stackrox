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

function FalsePositiveRequestActionsColumn({
    row,
    setRequestsToBeAssessed,
    canApproveRequest,
    canCancelRequest,
}): ReactElement {
    const items: IActions = [
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
            isDisabled: !canApproveRequest,
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
            isDisabled: !canApproveRequest,
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
            isDisabled: !canCancelRequest,
        },
    ];
    return <ActionsColumn items={items} />;
}

export default FalsePositiveRequestActionsColumn;
