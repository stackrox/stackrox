import React, { ReactElement } from 'react';
import { ActionsColumn } from '@patternfly/react-table';
import { FalsePositiveCVEsToBeAssessed } from './types';
import { Vulnerability } from '../imageVulnerabilities.graphql';

export type FalsePositiveCVEActionsColumnProps = {
    row: Vulnerability;
    setVulnsToBeAssessed: React.Dispatch<React.SetStateAction<FalsePositiveCVEsToBeAssessed>>;
    canReobserveCVE: boolean;
};

function FalsePositiveCVEActionsColumn({
    row,
    setVulnsToBeAssessed,
    canReobserveCVE,
}: FalsePositiveCVEActionsColumnProps): ReactElement {
    const items = [
        {
            title: 'Reobserve CVE',
            onClick: (event) => {
                event.preventDefault();
                setVulnsToBeAssessed({
                    type: 'FALSE_POSITIVE',
                    action: 'UNDO',
                    requestIDs: [row.vulnerabilityRequest.id],
                });
            },
            isDisabled: !canReobserveCVE,
        },
    ];
    return <ActionsColumn items={items} />;
}

export default FalsePositiveCVEActionsColumn;
