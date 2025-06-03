import React, { useCallback, useEffect, useState } from 'react';
import { Divider, Stack, StackItem, ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';

import { TimeWindow } from 'constants/timeWindows';
import useAnalytics, { DEPLOYMENT_FLOWS_TOGGLE_CLICKED } from 'hooks/useAnalytics';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import { UseUrlSearchReturn } from 'hooks/useURLSearch';

import { CustomNodeModel } from '../types/topology.type';
import { EdgeState } from '../components/EdgeStateSelect';
import { Flow } from '../types/flow.type';
import InternalFlows from './InternalFlows';
import ExternalFlows from './ExternalFlows';
import { isInternalFlow } from '../utils/networkGraphUtils';

export type DeploymentFlowsView = 'external-flows' | 'internal-flows';

type DeploymentFlowsProps = {
    deploymentId: string;
    nodes: CustomNodeModel[];
    edgeState: EdgeState;
    onNodeSelect: (id: string) => void;
    isLoadingNetworkFlows: boolean;
    networkFlowsError: string;
    networkFlows: Flow[];
    refetchFlows: () => void;
    anomalousUrlPagination: UseURLPaginationResult;
    baselineUrlPagination: UseURLPaginationResult;
    urlSearchFiltering: UseUrlSearchReturn;
    timeWindow: TimeWindow;
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
    anomalousUrlPagination,
    baselineUrlPagination,
    urlSearchFiltering,
    timeWindow,
}: DeploymentFlowsProps) {
    const { analyticsTrack } = useAnalytics();
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isNetworkGraphExternalIpsEnabled = isFeatureFlagEnabled('ROX_NETWORK_GRAPH_EXTERNAL_IPS');
    const [selectedView, setSelectedView] = useState<DeploymentFlowsView>('internal-flows');

    const { setPage: setPageAnomalous } = anomalousUrlPagination;
    const { setPage: setPageBaseline } = baselineUrlPagination;
    const { setSearchFilter } = urlSearchFiltering;

    useEffect(() => {
        setPageAnomalous(1);
        setPageBaseline(1);
        setSearchFilter({});
    }, [selectedView, setPageAnomalous, setPageBaseline, setSearchFilter]);

    // can be removed when routing is added to network graph
    const handleToggle = useCallback(
        (view: DeploymentFlowsView) => {
            if (view !== selectedView) {
                setSelectedView(view);

                const formattedView =
                    view === 'internal-flows' ? 'Internal Flows' : 'External Flows';

                analyticsTrack({
                    event: DEPLOYMENT_FLOWS_TOGGLE_CLICKED,
                    properties: { view: formattedView },
                });
            }
        },
        [selectedView, analyticsTrack]
    );

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
                            onChange={() => handleToggle('internal-flows')}
                        />
                        <ToggleGroupItem
                            text="External flows"
                            buttonId="external-flows"
                            isSelected={selectedView === 'external-flows'}
                            onChange={() => handleToggle('external-flows')}
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
                                networkFlows={networkFlows.filter((flow) => isInternalFlow(flow))}
                                refetchFlows={refetchFlows}
                            />
                        ) : (
                            <ExternalFlows
                                deploymentId={deploymentId}
                                timeWindow={timeWindow}
                                anomalousUrlPagination={anomalousUrlPagination}
                                baselineUrlPagination={baselineUrlPagination}
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
