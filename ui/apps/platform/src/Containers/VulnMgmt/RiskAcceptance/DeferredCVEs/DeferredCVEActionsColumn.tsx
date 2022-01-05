import React, { ReactElement } from 'react';
import { ActionsColumn } from '@patternfly/react-table';
import { DeferredCVEsToBeAssessed } from './types';
import { VulnerabilityWithRequest } from '../imageVulnerabilities.graphql';

export type DeferredCVEActionsColumnProps = {
    row: VulnerabilityWithRequest;
    setVulnsToBeAssessed: React.Dispatch<React.SetStateAction<DeferredCVEsToBeAssessed>>;
    canReobserveCVE: boolean;
};

function DeferredCVEActionsColumn({
    row,
    setVulnsToBeAssessed,
    canReobserveCVE,
}: DeferredCVEActionsColumnProps): ReactElement {
    const items = [
        {
            title: 'Reobserve CVE',
            onClick: (event) => {
                event.preventDefault();
                setVulnsToBeAssessed({
                    type: 'DEFERRAL',
                    action: 'UNDO',
                    requestIDs: [row.vulnerabilityRequest.id],
                });
            },
            isDisabled: !canReobserveCVE,
        },
    ];
    return <ActionsColumn items={items} />;
}

export default DeferredCVEActionsColumn;
