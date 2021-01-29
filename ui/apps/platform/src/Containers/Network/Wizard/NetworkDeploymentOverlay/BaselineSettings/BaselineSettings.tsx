import React, { ReactElement } from 'react';

import { filterModes } from 'constants/networkFilterModes';
import useFeatureFlagEnabled from 'hooks/useFeatureFlagEnabled';
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
    const isBaselineViolationEnabled = useFeatureFlagEnabled(
        'ROX_NETWORK_DETECTION_BASELINE_VIOLATION'
    );
    const {
        data: { networkBaselines, isAlertingEnabled },
        isLoading,
    } = useFetchNetworkBaselines({
        selectedDeployment,
        deploymentId,
        filterState,
    });

    const headerComponents = isBaselineViolationEnabled && (
        <AlertBaselineViolations deploymentId={deploymentId} isEnabled={isAlertingEnabled} />
    );

    return (
        <NetworkBaselines
            header="Baseline Settings"
            headerComponents={headerComponents}
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
