import React, { ReactElement } from 'react';
import pluralize from 'pluralize';

import { filterModes } from 'constants/networkFilterModes';
import useFetchBlockedFlows from './useFetchBlockedFlows';

import NetworkBaselines from '../NetworkBaselines';

export type BlockedFlowsProps = {
    selectedDeployment: unknown;
    deploymentId: string;
    filterState: string;
    onNavigateToEntity: () => void;
};

function getPanelHeaderText(numBlockedFlows: number): string {
    return `${numBlockedFlows} Blocked ${pluralize('Flow', numBlockedFlows)}`;
}

function BlockedFlows({
    selectedDeployment,
    deploymentId,
    filterState,
    onNavigateToEntity,
}: BlockedFlowsProps): ReactElement {
    const {
        data: { blockedFlows },
        isLoading,
    } = useFetchBlockedFlows({
        selectedDeployment,
        deploymentId,
        filterState,
    });

    const header = getPanelHeaderText(blockedFlows.length);

    return (
        <NetworkBaselines
            header={header}
            isLoading={isLoading}
            // TODO: might have to reconsider the name for this component since blocked flows != network baselines
            networkBaselines={blockedFlows}
            deploymentId={deploymentId}
            filterState={filterModes}
            onNavigateToEntity={onNavigateToEntity}
            showAnomalousFlows={false}
        />
    );
}

export default BlockedFlows;
