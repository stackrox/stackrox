import React, { ReactElement } from 'react';
import { ActionsColumn } from '@patternfly/react-table';
import { FalsePositiveCVEsToBeAssessed } from './types';
import { Vulnerability } from '../imageVulnerabilities.graphql';

export type FalsePositiveCVEActionsColumnProps = {
    row: Vulnerability;
    setVulnsToBeAssessed: React.Dispatch<React.SetStateAction<FalsePositiveCVEsToBeAssessed>>;
};

function FalsePositiveCVEActionsColumn({
    row,
    setVulnsToBeAssessed,
}: FalsePositiveCVEActionsColumnProps): ReactElement {
    const items = [
        {
            title: 'Reobserve CVE',
            onClick: (event) => {
                event.preventDefault();
                // @TODO: pass the vuln request id for this vuln in requestIDs
                setVulnsToBeAssessed({
                    type: 'FALSE_POSITIVE',
                    action: 'UNDO',
                    requestIDs: [row.id],
                });
            },
        },
    ];
    return <ActionsColumn items={items} />;
}

export default FalsePositiveCVEActionsColumn;
