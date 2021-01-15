import React, { ReactElement } from 'react';

import { filterModes } from 'constants/networkFilterModes';
import useFetchNetworkBaselines from './useFetchNetworkBaselines';

import NetworkBaselines from '../NetworkBaselines';

export type BaselineSettingsProps = {
    selectedDeployment: unknown;
    deploymentId: string;
    filterState: string;
    onNavigateToEntity: () => void;
};

function BaselineSettings({
    selectedDeployment,
    deploymentId,
    filterState,
    onNavigateToEntity,
}: BaselineSettingsProps): ReactElement {
    const { data: networkBaselines, isLoading } = useFetchNetworkBaselines({
        selectedDeployment,
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
            onNavigateToEntity={onNavigateToEntity}
            showAnomalousFlows={false}
        />
    );
}

export default BaselineSettings;
