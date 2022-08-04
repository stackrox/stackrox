import React, { ReactElement } from 'react';

import useNavigateToEntity from 'Containers/Network/SidePanel/useNavigateToEntity';
import { FilterState, NetworkNode, Edge } from 'Containers/Network/networkTypes';

import Tab from 'Components/Tab';
import BinderTabs from 'Components/BinderTabs';
import NetworkFlows from './NetworkFlows';
import BaselineSettings from './BaselineSettings';

export type FlowsProps = {
    deploymentId: string;
    selectedDeployment: NetworkNode;
    filterState: FilterState;
    lastUpdatedTimestamp: string;
    entityIdToNamespaceMap: Record<string, string>;
};

function filterNonSelfReferencingEdges(edge: Edge) {
    const { destNodeName, destNodeNamespace, source, target } = edge.data;
    return destNodeName && destNodeNamespace && source !== target;
}

function Flows({
    deploymentId,
    selectedDeployment,
    filterState,
    lastUpdatedTimestamp,
    entityIdToNamespaceMap,
}: FlowsProps): ReactElement {
    const edges = selectedDeployment.edges.filter(filterNonSelfReferencingEdges);
    const onNavigateToEntity = useNavigateToEntity();

    return (
        <BinderTabs>
            <Tab title="Network Flows">
                <NetworkFlows
                    deploymentId={deploymentId}
                    edges={edges}
                    filterState={filterState}
                    onNavigateToEntity={onNavigateToEntity}
                    lastUpdatedTimestamp={lastUpdatedTimestamp}
                />
            </Tab>
            <Tab title="Baseline Settings">
                <BaselineSettings
                    selectedDeployment={selectedDeployment}
                    deploymentId={deploymentId}
                    filterState={filterState}
                    onNavigateToEntity={onNavigateToEntity}
                    entityIdToNamespaceMap={entityIdToNamespaceMap}
                />
            </Tab>
        </BinderTabs>
    );
}

export default Flows;
