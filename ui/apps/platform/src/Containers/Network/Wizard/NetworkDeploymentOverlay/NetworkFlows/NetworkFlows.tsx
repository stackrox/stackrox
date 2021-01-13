import React, { ReactElement } from 'react';
import pluralize from 'pluralize';

import { filterModes } from 'constants/networkFilterModes';
import useFetchNetworkBaselines from './useFetchNetworkBaselines';

import NetworkBaselines from '../NetworkBaselines';

function getPanelHeaderText(numBaselineFlows, filterState): string {
    switch (filterState) {
        case filterModes.active:
            return `${numBaselineFlows} active ${pluralize('flow', numBaselineFlows)}`;
        case filterModes.allowed:
            return `${numBaselineFlows} allowed ${pluralize('flow', numBaselineFlows)}`;
        default:
            return `${numBaselineFlows} ${pluralize('flow', numBaselineFlows)}`;
    }
}

function NetworkFlows({
    deploymentId,
    edges,
    filterState,
    onNavigateToEntity,
    lastUpdatedTimestamp,
}): ReactElement {
    const { data: networkBaselines, isLoading } = useFetchNetworkBaselines({
        deploymentId,
        edges,
        filterState,
        lastUpdatedTimestamp,
    });

    const header = getPanelHeaderText(networkBaselines.length, filterState);

    return (
        <NetworkBaselines
            header={header}
            isLoading={isLoading}
            networkBaselines={networkBaselines}
            deploymentId={deploymentId}
            filterState={filterModes}
            onNavigateToEntity={onNavigateToEntity}
        />
    );
}

export default NetworkFlows;
