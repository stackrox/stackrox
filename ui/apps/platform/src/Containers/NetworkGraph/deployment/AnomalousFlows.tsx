import React from 'react';
import { FlexItem, Label } from '@patternfly/react-core';
import { ExclamationCircleIcon, ExclamationTriangleIcon } from '@patternfly/react-icons';
import pluralize from 'pluralize';

import { Flow } from '../types/flow.type';
import {
    getNumAnomalousExternalFlows,
    getNumAnomalousInternalFlows,
} from '../utils/networkGraphUtils';

export type AnomalousFlowsProps = {
    networkFlows: Flow[];
};

function AnomalousFlows({ networkFlows }: AnomalousFlowsProps) {
    const numAnomalousExternalFlows = getNumAnomalousExternalFlows(networkFlows);
    const numAnomalousInternalFlows = getNumAnomalousInternalFlows(networkFlows);

    if (numAnomalousExternalFlows === 0 && numAnomalousInternalFlows === 0) {
        return <>None</>;
    }

    return (
        <>
            {numAnomalousExternalFlows !== 0 && (
                <FlexItem>
                    <Label variant="outline" color="red" icon={<ExclamationCircleIcon />}>
                        {numAnomalousExternalFlows} external{' '}
                        {pluralize('flow', numAnomalousExternalFlows)}
                    </Label>
                </FlexItem>
            )}
            {numAnomalousInternalFlows !== 0 && (
                <FlexItem>
                    <Label variant="outline" color="gold" icon={<ExclamationTriangleIcon />}>
                        {numAnomalousInternalFlows} internal{' '}
                        {pluralize('flow', numAnomalousInternalFlows)}
                    </Label>
                </FlexItem>
            )}
        </>
    );
}

export default AnomalousFlows;
