import React from 'react';
import { Divider, Stack, StackItem, ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';

import useFeatureFlags from 'hooks/useFeatureFlags';

import { CustomNodeModel } from '../types/topology.type';
import { EdgeState } from '../components/EdgeStateSelect';
import { Flow } from '../types/flow.type';
import InternalFlows from './InternalFlows';
import ExternalFlows from './ExternalFlows';

import {
    usePaginationAnomalous,
    usePaginationBaseline,
    useSearchFilterSidePanel,
    useParameterSidePanelToggle,
} from '../URLStateContext';

export type DeploymentFlowsView = 'EXTERNAL_FLOWS' | 'INTERNAL_FLOWS';

type DeploymentFlowsProps = {
    deploymentId: string;
    nodes: CustomNodeModel[];
    edgeState: EdgeState;
    onNodeSelect: (id: string) => void;
    isLoadingNetworkFlows: boolean;
    networkFlowsError: string;
    networkFlows: Flow[];
    refetchFlows: () => void;
};

function DeploymentFlows({
    deploymentId,
    nodes,
    edgeState,
    onNodeSelect,
    isLoadingNetworkFlows,
    networkFlowsError,
    networkFlows,
    refetchFlows,
}: DeploymentFlowsProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isNetworkGraphExternalIpsEnabled = isFeatureFlagEnabled('ROX_NETWORK_GRAPH_EXTERNAL_IPS');

    const { setPage: setPageAnomalous } = usePaginationAnomalous();
    const { setPage: setPageBaseline } = usePaginationBaseline();
    const { setSearchFilter } = useSearchFilterSidePanel();
    const { selectedToggleSidePanel, setSelectedToggleSidePanel } = useParameterSidePanelToggle();

    function changeView(view: DeploymentFlowsView) {
        setSelectedToggleSidePanel(view);
        setPageAnomalous(1);
        setPageBaseline(1);
        setSearchFilter({});
    }

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

    const selectedView = selectedToggleSidePanel ?? 'INTERNAL_FLOWS';

    return (
        <div className="pf-v5-u-h-100">
            <Stack>
                <StackItem className="pf-v5-u-p-md">
                    <ToggleGroup aria-label="Toggle between internal flows and external flows views">
                        <ToggleGroupItem
                            text="Internal flows"
                            buttonId="INTERNAL_FLOWS"
                            isSelected={selectedView === 'INTERNAL_FLOWS'}
                            onChange={() => changeView('INTERNAL_FLOWS')}
                        />
                        <ToggleGroupItem
                            text="External flows"
                            buttonId="EXTERNAL_FLOWS"
                            isSelected={selectedView === 'EXTERNAL_FLOWS'}
                            onChange={() => changeView('EXTERNAL_FLOWS')}
                        />
                    </ToggleGroup>
                </StackItem>
                <Divider component="hr" />
                <StackItem isFilled style={{ overflow: 'auto' }}>
                    <Stack className="pf-v5-u-p-md">
                        {selectedView === 'INTERNAL_FLOWS' ? (
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
                            <ExternalFlows deploymentId={deploymentId} />
                        )}
                    </Stack>
                </StackItem>
            </Stack>
        </div>
    );
}

export default DeploymentFlows;
