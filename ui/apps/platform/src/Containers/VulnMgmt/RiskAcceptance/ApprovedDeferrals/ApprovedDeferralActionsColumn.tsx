import React, { ReactElement } from 'react';
import { ActionsColumn, IActions } from '@patternfly/react-table';
import { VulnerabilityRequest } from '../vulnerabilityRequests.graphql';
import { ApprovedDeferralRequestsToBeAssessed } from './types';

export type ApprovedDeferralActionsColumnProps = {
    row: VulnerabilityRequest;
    setRequestsToBeAssessed: React.Dispatch<
        React.SetStateAction<ApprovedDeferralRequestsToBeAssessed>
    >;
    canUpdateDeferral: boolean;
    canReobserveCVE: boolean;
};

function ApprovedDeferralActionsColumn({
    row,
    setRequestsToBeAssessed,
    canUpdateDeferral,
    canReobserveCVE,
}: ApprovedDeferralActionsColumnProps): ReactElement {
    const items: IActions = [
        {
            title: 'Update deferral',
            onClick: (event) => {
                event.preventDefault();
                setRequestsToBeAssessed({
                    type: 'DEFERRAL',
                    action: 'UPDATE',
                    requestIDs: [row.id],
                });
            },
            isDisabled: !canUpdateDeferral,
        },
        {
            title: 'Reobserve CVE',
            onClick: (event) => {
                event.preventDefault();
                setRequestsToBeAssessed({
                    type: 'DEFERRAL',
                    action: 'UNDO',
                    requestIDs: [row.id],
                });
            },
            isDisabled: !canReobserveCVE,
        },
    ];
    return <ActionsColumn items={items} />;
}

export default ApprovedDeferralActionsColumn;
