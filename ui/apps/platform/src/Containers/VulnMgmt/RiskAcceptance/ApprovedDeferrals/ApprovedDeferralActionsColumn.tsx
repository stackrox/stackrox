import React, { ReactElement } from 'react';
import { ActionsColumn } from '@patternfly/react-table';
import { VulnerabilityRequest } from '../vulnerabilityRequests.graphql';
import { ApprovedDeferralRequestsToBeAssessed } from './types';

export type ApprovedDeferralActionsColumnProps = {
    row: VulnerabilityRequest;
    setRequestsToBeAssessed: React.Dispatch<
        React.SetStateAction<ApprovedDeferralRequestsToBeAssessed>
    >;
};

function ApprovedDeferralActionsColumn({
    row,
    setRequestsToBeAssessed,
}: ApprovedDeferralActionsColumnProps): ReactElement {
    const items = [
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
        },
        {
            title: 'Reobserve CVE',
            onClick: (event) => {
                event.preventDefault();
                setRequestsToBeAssessed({ type: 'DEFERRAL', action: 'UNDO', requestIDs: [row.id] });
            },
        },
    ];
    return <ActionsColumn items={items} />;
}

export default ApprovedDeferralActionsColumn;
