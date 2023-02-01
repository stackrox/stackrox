import React, { useEffect, useState, useCallback, useRef } from 'react';
import { useHistory } from 'react-router-dom';
import { Popover } from '@patternfly/react-core';
import {
    SELECTION_EVENT,
    SelectionEventListener,
    useEventListener,
    TopologySideBar,
    TopologyView,
    createTopologyControlButtons,
    defaultControlButtonsOptions,
    TopologyControlBar,
    useVisualizationController,
    VisualizationSurface,
} from '@patternfly/react-topology';

import { networkBasePathPF } from 'routePaths';
import { getQueryObject, getQueryString } from 'utils/queryStringUtils';
import DeploymentSideBar from './deployment/DeploymentSideBar';
import NamespaceSideBar from './namespace/NamespaceSideBar';
import CidrBlockSideBar from './cidr/CidrBlockSideBar';
import ExternalEntitiesSideBar from './externalEntities/ExternalEntitiesSideBar';
import ExternalGroupSideBar from './external/ExternalGroupSideBar';
import NetworkPolicySimulatorSidePanel from './simulation/NetworkPolicySimulatorSidePanel';
import { getNodeById } from './utils/networkGraphUtils';
import { CustomModel, CustomNodeModel } from './types/topology.type';
import { Simulation } from './utils/getSimulation';
import LegendContent from './components/LegendContent';

import {
    ApplyNetworkPolicyModification,
    NetworkPolicySimulator,
    SetNetworkPolicyModification,
} from './hooks/useNetworkPolicySimulator';
import { EdgeState } from './components/EdgeStateSelect';
import { deploymentTabs } from './utils/deploymentUtils';

// TODO: move these type defs to a central location
export const UrlDetailType = {
    NAMESPACE: 'namespace',
    DEPLOYMENT: 'deployment',
    CIDR_BLOCK: 'cidr',
    EXTERNAL_ENTITIES: 'internet',
    EXTERNAL_GROUP: 'external',
} as const;
export type UrlDetailTypeKey = keyof typeof UrlDetailType;
export type UrlDetailTypeValue = typeof UrlDetailType[UrlDetailTypeKey];

function getUrlParamsForEntity(type, id): [UrlDetailTypeValue, string] {
    // eslint-disable-next-line @typescript-eslint/no-unsafe-return
    return [UrlDetailType[type], id];
}

export type TopologyComponentProps = {
    model: CustomModel;
    simulation: Simulation;
    selectedClusterId: string;
    selectedNode?: CustomNodeModel;
    simulator: NetworkPolicySimulator;
    setNetworkPolicyModification: SetNetworkPolicyModification;
    applyNetworkPolicyModification: ApplyNetworkPolicyModification;
    edgeState: EdgeState;
};

// @TODO: Consider a better approach to managing the side panel related state (simulation + URL path for entities)
function clearSimulationQuery(search: string): string {
    const modifiedSearchFilter = getQueryObject(search);
    delete modifiedSearchFilter.simulation;
    const queryString = getQueryString(modifiedSearchFilter);
    return queryString;
}

const TopologyComponent = ({
    model,
    simulation,
    selectedClusterId,
    selectedNode,
    simulator,
    setNetworkPolicyModification,
    applyNetworkPolicyModification,
    edgeState,
}: TopologyComponentProps) => {
    const firstRenderRef = useRef(true);
    const history = useHistory();
    const controller = useVisualizationController();
    const [defaultDeploymentTab, setDefaultDeploymentTab] = useState(deploymentTabs.DETAILS);

    function closeSidebar() {
        const queryString = clearSimulationQuery(history.location.search);
        history.push(`${networkBasePathPF}${queryString}`);
    }

    function onNodeClick(ids: string[]) {
        const newSelectedId = ids?.[0] || '';
        const newSelectedEntity = getNodeById(model?.nodes, newSelectedId);
        if (selectedNode && !newSelectedId) {
            closeSidebar();
        } else if (newSelectedEntity?.data.type === 'EXTRANEOUS') {
            setDefaultDeploymentTab(deploymentTabs.FLOWS);
        } else if (newSelectedEntity) {
            setDefaultDeploymentTab(deploymentTabs.DETAILS);
            const { data, id } = newSelectedEntity;
            const [newDetailType, newDetailId] = getUrlParamsForEntity(data.type, id);
            const queryString = clearSimulationQuery(history.location.search);
            // if found, and it's not the logical grouping of all external sources, then trigger URL update
            if (newDetailId !== 'EXTERNAL') {
                history.push(`${networkBasePathPF}/${newDetailType}/${newDetailId}${queryString}`);
            } else {
                // otherwise, return to the graph-only state
                history.push(`${networkBasePathPF}${queryString}`);
            }
        }
    }

    function onNodeSelect(id: string) {
        onNodeClick([id]);
    }

    function zoomInCallback() {
        controller.getGraph().scaleBy(4 / 3);
    }

    function zoomOutCallback() {
        controller.getGraph().scaleBy(0.75);
    }

    const fitToScreenCallback = useCallback(() => {
        controller.getGraph().fit(80);
    }, [controller]);

    const resetViewCallback = useCallback(() => {
        controller.getGraph().reset();
        controller.getGraph().layout();
    }, [controller]);

    useEventListener<SelectionEventListener>(SELECTION_EVENT, (ids) => {
        onNodeClick(ids);
    });

    useEffect(() => {
        controller.fromModel(model);
        if (selectedNode) {
            const selectedNodeElement = controller.getNodeById(selectedNode.id);
            if (selectedNodeElement) {
                // the offset is to make sure the label also makes it inside the viewport
                controller.getGraph().panIntoView(selectedNodeElement, { offset: 50 });
            }
        }
    }, [controller, model, selectedNode]);

    useEffect(() => {
        // we don't want to reset view on init
        if (!firstRenderRef.current) {
            resetViewCallback();
        } else {
            firstRenderRef.current = false;
        }
    }, [edgeState, resetViewCallback, fitToScreenCallback]);

    const selectedIds = selectedNode ? [selectedNode.id] : [];

    return (
        <TopologyView
            sideBar={
                <TopologySideBar resizable onClose={closeSidebar}>
                    {simulation.isOn && simulation.type === 'networkPolicy' && (
                        <NetworkPolicySimulatorSidePanel
                            selectedClusterId={selectedClusterId}
                            simulator={simulator}
                            setNetworkPolicyModification={setNetworkPolicyModification}
                            applyNetworkPolicyModification={applyNetworkPolicyModification}
                        />
                    )}
                    {selectedNode && selectedNode?.data?.type === 'NAMESPACE' && (
                        <NamespaceSideBar
                            namespaceId={selectedNode.id}
                            nodes={model?.nodes || []}
                            edges={model?.edges || []}
                            onNodeSelect={onNodeSelect}
                        />
                    )}
                    {selectedNode && selectedNode?.data?.type === 'DEPLOYMENT' && (
                        <DeploymentSideBar
                            deploymentId={selectedNode.id}
                            nodes={model?.nodes || []}
                            edges={model?.edges || []}
                            edgeState={edgeState}
                            onNodeSelect={onNodeSelect}
                            defaultDeploymentTab={defaultDeploymentTab}
                        />
                    )}
                    {selectedNode && selectedNode?.data?.type === 'EXTERNAL_GROUP' && (
                        <ExternalGroupSideBar
                            id={selectedNode.id}
                            nodes={model?.nodes || []}
                            edges={model?.edges || []}
                            onNodeSelect={onNodeSelect}
                        />
                    )}
                    {selectedNode && selectedNode?.data?.type === 'CIDR_BLOCK' && (
                        <CidrBlockSideBar
                            id={selectedNode.id}
                            nodes={model?.nodes || []}
                            edges={model?.edges || []}
                            onNodeSelect={onNodeSelect}
                        />
                    )}
                    {selectedNode && selectedNode?.data?.type === 'EXTERNAL_ENTITIES' && (
                        <ExternalEntitiesSideBar
                            id={selectedNode.id}
                            nodes={model?.nodes || []}
                            edges={model?.edges || []}
                            onNodeSelect={onNodeSelect}
                        />
                    )}
                </TopologySideBar>
            }
            sideBarOpen={!!selectedNode || simulation.isOn}
            sideBarResizable
            controlBar={
                <TopologyControlBar
                    controlButtons={createTopologyControlButtons({
                        ...defaultControlButtonsOptions,
                        zoomInCallback,
                        zoomOutCallback,
                        fitToScreenCallback,
                        resetViewCallback,
                    })}
                />
            }
        >
            <VisualizationSurface state={{ selectedIds }} />
            <Popover
                aria-label="Network graph legend"
                bodyContent={<LegendContent />}
                hasAutoWidth
                reference={() => document.getElementById('legend') as HTMLButtonElement}
            />
        </TopologyView>
    );
};

export default TopologyComponent;
