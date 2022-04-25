import React, { ReactElement } from 'react';

import { NavigateToEntityHook } from 'Containers/Network/SidePanel/useNavigateToEntity';
import { filterModes } from 'constants/networkFilterModes';
import { FilterState } from 'Containers/Network/networkTypes';
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
    const {
        data: { networkBaselines, isAlertingEnabled },
        isLoading,
    } = useFetchNetworkBaselines({
        selectedDeployment,
        deploymentId,
        filterState,
        entityIdToNamespaceMap,
    });

    const headerComponents = (
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
                excludedSearchCategories={['State']}
            />
            <div className="flex justify-center items-center py-4 border-t border-base-300 bg-base-100">
                <SimulateBaselineNetworkPolicy />
            </div>
        </div>
    );
}

export default BaselineSettings;
