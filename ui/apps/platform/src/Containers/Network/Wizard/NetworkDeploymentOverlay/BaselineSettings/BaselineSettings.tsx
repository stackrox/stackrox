import React, { ReactElement } from 'react';

import { filterModes } from 'constants/networkFilterModes';
import useFetchNetworkBaselines from './useFetchNetworkBaselines';

import NetworkBaselines from '../NetworkBaselines';
import AlertBaselineViolations from './AlertBaselineViolations';

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
    const {
        data: { networkBaselines, isAlertingEnabled },
        isLoading,
    } = useFetchNetworkBaselines({
        selectedDeployment,
        deploymentId,
        filterState,
    });

    return (
        <NetworkBaselines
            header="Baseline Settings"
            headerComponents={
                <AlertBaselineViolations
                    deploymentId={deploymentId}
                    isEnabled={isAlertingEnabled}
                />
            }
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
