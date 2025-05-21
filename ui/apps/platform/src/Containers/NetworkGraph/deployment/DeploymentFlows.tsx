import React, { useEffect, useState } from 'react';
import { Divider, Stack, StackItem, ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';

import { TimeWindow } from 'constants/timeWindows';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import { UseUrlSearchReturn } from 'hooks/useURLSearch';

import { CustomNodeModel } from '../types/topology.type';
import { EdgeState } from '../components/EdgeStateSelect';
import { Flow } from '../types/flow.type';
import InternalFlows from './InternalFlows';
import ExternalFlows from './ExternalFlows';

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
                                deploymentId={deploymentId}
                                timeWindow={timeWindow}
                                anomalousUrlPagination={anomalousUrlPagination}
                                baselineUrlPagination={baselineUrlPagination}
                            />
                        )}
                    </Stack>
                </StackItem>
            </Stack>
        </div>
    );
}

export default DeploymentFlows;
