import React, { useCallback, useEffect } from 'react';
import { Divider, Stack, StackItem, ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';

import useAnalytics, { DEPLOYMENT_FLOWS_TOGGLE_CLICKED } from 'hooks/useAnalytics';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { QueryValue } from 'hooks/useURLParameter';

import useFetchNetworkFlows from '../api/useFetchNetworkFlows';
import { EdgeState } from '../components/EdgeStateSelect';
import { CustomEdgeModel, CustomNodeModel } from '../types/topology.type';
import { isInternalFlow } from '../utils/networkGraphUtils';

import InternalFlows from './InternalFlows';
import ExternalFlows from './ExternalFlows';

import {
    usePagination,
    usePaginationSecondary,
    useSearchFilterSidePanel,
    useSidePanelToggle,
} from '../NetworkGraphURLStateContext';

export type DeploymentFlowsView = 'EXTERNAL_FLOWS' | 'INTERNAL_FLOWS';

const DEPLOYMENT_FLOWS_TOGGLES = ['INTERNAL_FLOWS', 'EXTERNAL_FLOWS'] as const;
export type DeploymentFlowsToggleKey = (typeof DEPLOYMENT_FLOWS_TOGGLES)[number];

export const DEFAULT_DEPLOYMENT_FLOWS_TOGGLE: DeploymentFlowsToggleKey = 'INTERNAL_FLOWS';

export function isValidDeploymentFlowsToggle(value: QueryValue): value is DeploymentFlowsToggleKey {
    return typeof value === 'string' && DEPLOYMENT_FLOWS_TOGGLES.some((state) => state === value);
}

type DeploymentFlowsProps = {
    deploymentId: string;
    edgeState: EdgeState;
    edges: CustomEdgeModel[];
    nodes: CustomNodeModel[];
    onNodeSelect: (id: string) => void;
};

function DeploymentFlows({
    deploymentId,
    nodes,
    edgeState,
    edges,
    onNodeSelect,
}: DeploymentFlowsProps) {
    const { analyticsTrack } = useAnalytics();
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isNetworkGraphExternalIpsEnabled = isFeatureFlagEnabled('ROX_NETWORK_GRAPH_EXTERNAL_IPS');

    const { setPage: setPageAnomalous } = usePagination();
    const { setPage: setPageBaseline } = usePaginationSecondary();
    const { setSearchFilter } = useSearchFilterSidePanel();
    const { selectedToggleSidePanel, setSelectedToggleSidePanel } = useSidePanelToggle();

    useEffect(() => {
        if (
            selectedToggleSidePanel !== undefined &&
            !isValidDeploymentFlowsToggle(selectedToggleSidePanel)
        ) {
            setSelectedToggleSidePanel(DEFAULT_DEPLOYMENT_FLOWS_TOGGLE, 'replace');
        }
    }, [selectedToggleSidePanel, setSelectedToggleSidePanel]);

    const handleToggle = useCallback(
        (view: DeploymentFlowsView) => {
            if (view !== selectedToggleSidePanel) {
                setSelectedToggleSidePanel(view);
                setPageAnomalous(1);
                setPageBaseline(1);
                setSearchFilter({});

                const formattedView =
                    view === 'INTERNAL_FLOWS' ? 'Internal Flows' : 'External Flows';

                analyticsTrack({
                    event: DEPLOYMENT_FLOWS_TOGGLE_CLICKED,
                    properties: { view: formattedView },
                });
            }
        },
        [
            analyticsTrack,
            selectedToggleSidePanel,
            setPageAnomalous,
            setPageBaseline,
            setSearchFilter,
            setSelectedToggleSidePanel,
        ]
    );

    const {
        isLoading: isLoadingNetworkFlows,
        error: networkFlowsError,
        data: { networkFlows },
        refetchFlows,
    } = useFetchNetworkFlows({ deploymentId, edgeState, edges, nodes });

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

    const selectedView: DeploymentFlowsToggleKey = isValidDeploymentFlowsToggle(
        selectedToggleSidePanel
    )
        ? selectedToggleSidePanel
        : DEFAULT_DEPLOYMENT_FLOWS_TOGGLE;

    return (
        <div className="pf-v5-u-h-100">
            <Stack>
                <StackItem className="pf-v5-u-p-md">
                    <ToggleGroup aria-label="Toggle between internal flows and external flows views">
                        <ToggleGroupItem
                            text="Internal flows"
                            buttonId="internal-flows"
                            isSelected={selectedView === 'INTERNAL_FLOWS'}
                            onChange={() => handleToggle('INTERNAL_FLOWS')}
                        />
                        <ToggleGroupItem
                            text="External flows"
                            buttonId="external-flows"
                            isSelected={selectedView === 'EXTERNAL_FLOWS'}
                            onChange={() => handleToggle('EXTERNAL_FLOWS')}
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
                                networkFlows={networkFlows.filter((flow) => isInternalFlow(flow))}
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
