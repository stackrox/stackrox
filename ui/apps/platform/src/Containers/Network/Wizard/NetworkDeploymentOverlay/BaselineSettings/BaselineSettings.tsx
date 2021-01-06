import React, { ReactElement } from 'react';

import { filterModes } from 'constants/networkFilterModes';
import useFetchNetworkBaselines from './useFetchNetworkBaselines';

import NetworkBaselines from '../NetworkBaselines';

function BaselineSettings({ deploymentId, filterState, onNavigateToDeploymentById }): ReactElement {
    const { data: networkBaselines, isLoading } = useFetchNetworkBaselines({
        deploymentId,
        filterState,
    });

    return (
        <NetworkBaselines
            header="Baseline Settings"
            isLoading={isLoading}
            networkBaselines={networkBaselines}
            deploymentId={deploymentId}
            filterState={filterModes}
            onNavigateToDeploymentById={onNavigateToDeploymentById}
        />
    );
}

export default BaselineSettings;
