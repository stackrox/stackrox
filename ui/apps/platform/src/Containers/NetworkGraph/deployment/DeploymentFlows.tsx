import React, { useEffect, useState } from 'react';
import { Divider, Stack, StackItem, ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import { UseUrlSearchReturn } from 'hooks/useURLSearch';

import { CustomNodeModel } from '../types/topology.type';
import { EdgeState } from '../components/EdgeStateSelect';
import { Flow } from '../types/flow.type';
import InternalFlows from './InternalFlows';
import ExternalFlows from './ExternalFlows';
import { NetworkScopeHierarchy } from '../types/networkScopeHierarchy';

export type DeploymentFlowsView = 'external-flows' | 'internal-flows';

type DeploymentFlowsProps = {
    deploymentId: string;
    deploymentName: string;
    nodes: CustomNodeModel[];
    edgeState: EdgeState;
    onNodeSelect: (id: string) => void;
    onExternalIPSelect: (externalIP: string) => void;
    isLoadingNetworkFlows: boolean;
    networkFlowsError: string;
    networkFlows: Flow[];
    refetchFlows: () => void;
    scopeHierarchy: NetworkScopeHierarchy;
    urlPagination: UseURLPaginationResult;
    urlSearchFiltering: UseUrlSearchReturn;
};

function DeploymentFlows({
    deploymentId,
    deploymentName,
    nodes,
    edgeState,
    onNodeSelect,
    onExternalIPSelect,
    isLoadingNetworkFlows,
    networkFlowsError,
    networkFlows,
    refetchFlows,
    scopeHierarchy,
    urlPagination,
    urlSearchFiltering,
}: DeploymentFlowsProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isNetworkGraphExternalIpsEnabled = isFeatureFlagEnabled('ROX_NETWORK_GRAPH_EXTERNAL_IPS');
    const [selectedView, setSelectedView] = useState<DeploymentFlowsView>('internal-flows');

    const { setPage } = urlPagination;
    const { setSearchFilter } = urlSearchFiltering;

    useEffect(() => {
        setPage(1);
        setSearchFilter({});
    }, [selectedView, setPage, setSearchFilter]);

    if (!isNetworkGraphExternalIpsEnabled) {
        return (
            <div className="pf-v5-u-h-100 pf-v5-u-p-md">
                <InternalFlows
                    nodes={nodes}
                    deploymentId={deploymentId}
                    edgeState={edgeState}
                    onNodeSelect={onNodeSelect}
                    isLoadingNetworkFlows={isLoadingNetworkFlows}
                    networkFlowsError={networkFlowsError}
                    networkFlows={networkFlows}
                    refetchFlows={refetchFlows}
                />
            </div>
        );
    }

    return (
        <div className="pf-v5-u-h-100">
            <Stack>
                <StackItem className="pf-v5-u-p-md">
                    <ToggleGroup aria-label="Toggle between internal flows and external flows views">
                        <ToggleGroupItem
                            text="Internal flows"
                            buttonId="internal-flows"
                            isSelected={selectedView === 'internal-flows'}
                            onChange={() => setSelectedView('internal-flows')}
                        />
                        <ToggleGroupItem
                            text="External flows"
                            buttonId="external-flows"
                            isSelected={selectedView === 'external-flows'}
                            onChange={() => setSelectedView('external-flows')}
                        />
                    </ToggleGroup>
                </StackItem>
                <Divider component="hr" />
                <StackItem isFilled style={{ overflow: 'auto' }}>
                    <Stack className="pf-v5-u-p-md">
                        {selectedView === 'internal-flows' ? (
                            <InternalFlows
                                nodes={nodes}
                                deploymentId={deploymentId}
                                edgeState={edgeState}
                                onNodeSelect={onNodeSelect}
                                isLoadingNetworkFlows={isLoadingNetworkFlows}
                                networkFlowsError={networkFlowsError}
                                networkFlows={networkFlows}
                                refetchFlows={refetchFlows}
                            />
                        ) : (
                            <ExternalFlows
                                deploymentName={deploymentName}
                                scopeHierarchy={scopeHierarchy}
                                onExternalIPSelect={onExternalIPSelect}
                                urlPagination={urlPagination}
                                urlSearchFiltering={urlSearchFiltering}
                            />
                        )}
                    </Stack>
                </StackItem>
            </Stack>
        </div>
    );
}

export default DeploymentFlows;
