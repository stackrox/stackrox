import React, { ReactElement } from 'react';
import { ActionsColumn } from '@patternfly/react-table';
import { DeferredCVEsToBeAssessed } from './types';
import { Vulnerability } from '../imageVulnerabilities.graphql';

export type DeferredCVEActionsColumnProps = {
    row: Vulnerability;
    setVulnsToBeAssessed: React.Dispatch<React.SetStateAction<DeferredCVEsToBeAssessed>>;
};

function DeferredCVEActionsColumn({
    row,
    setVulnsToBeAssessed,
}: DeferredCVEActionsColumnProps): ReactElement {
    const items = [
        {
            title: 'Reobserve CVE',
            onClick: (event) => {
                event.preventDefault();
                // @TODO: pass the vuln request id for this vuln in requestIDs
                setVulnsToBeAssessed({ type: 'DEFERRAL', action: 'UNDO', requestIDs: [row.id] });
            },
        },
    ];
    return <ActionsColumn items={items} />;
}

export default DeferredCVEActionsColumn;
