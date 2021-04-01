import React, { ReactElement } from 'react';

import { NavigateToEntityHook } from 'Containers/Network/Wizard/useNavigateToEntity';
import { filterModes } from 'constants/networkFilterModes';
import { FilterState } from 'Containers/Network/networkTypes';
import useFeatureFlagEnabled from 'hooks/useFeatureFlagEnabled';
import { knownBackendFlags } from 'utils/featureFlags';
import { networkFlowStatus } from 'constants/networkGraph';
import useFetchNetworkBaselines from './useFetchNetworkBaselines';

import NetworkBaselines from '../NetworkBaselines';
import AlertBaselineViolations from './AlertBaselineViolations';
import SimulateBaselineNetworkPolicy from './SimulateBaselineNetworkPolicy';

export type BaselineSettingsProps = {
    selectedDeployment: unknown;
    deploymentId: string;
    filterState: FilterState;
    onNavigateToEntity: NavigateToEntityHook;
    entityIdToNamespaceMap: Record<string, string>;
};

function BaselineSettings({
    selectedDeployment,
    entityIdToNamespaceMap,
    deploymentId,
    filterState,
    onNavigateToEntity,
}: BaselineSettingsProps): ReactElement {
    const isBaselineSimulationFeatureEnabled = useFeatureFlagEnabled(
        knownBackendFlags.ROX_NETWORK_DETECTION_BASELINE_SIMULATION
    );
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
        entityIdToNamespaceMap,
    });

    const headerComponents = isBaselineViolationEnabled && (
        <AlertBaselineViolations deploymentId={deploymentId} isEnabled={isAlertingEnabled} />
    );

    return (
        <div className="flex flex-1 flex-col">
            <NetworkBaselines
                header="Baseline Settings"
                headerComponents={headerComponents}
                isLoading={isLoading}
                networkBaselines={networkBaselines}
                deploymentId={deploymentId}
                filterState={filterModes}
                onNavigateToEntity={onNavigateToEntity}
                includedBaselineStatuses={[networkFlowStatus.BASELINE]}
            />
            {isBaselineSimulationFeatureEnabled && (
                <div className="flex justify-center items-center py-4 border-t border-base-300">
                    <SimulateBaselineNetworkPolicy />
                </div>
            )}
        </div>
    );
}

export default BaselineSettings;
