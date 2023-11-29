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

import { networkBasePath } from 'routePaths';
import useFetchDeploymentCount from 'hooks/useFetchDeploymentCount';
import usePermissions from 'hooks/usePermissions';
import DeploymentSideBar from './deployment/DeploymentSideBar';
import NamespaceSideBar from './namespace/NamespaceSideBar';
import CidrBlockSideBar from './cidr/CidrBlockSideBar';
import ExternalEntitiesSideBar from './externalEntities/ExternalEntitiesSideBar';
import ExternalGroupSideBar from './external/ExternalGroupSideBar';
import NetworkPolicySimulatorSidePanel, {
    clearSimulationQuery,
} from './simulation/NetworkPolicySimulatorSidePanel';
import { getNodeById } from './utils/networkGraphUtils';
import { CustomModel, CustomNodeModel } from './types/topology.type';
import { Simulation } from './utils/getSimulation';
import LegendContent from './components/LegendContent';

import {
    NetworkPolicySimulator,
    SetNetworkPolicyModification,
} from './hooks/useNetworkPolicySimulator';
import { EdgeState } from './components/EdgeStateSelect';
import { deploymentTabs } from './utils/deploymentUtils';
import { getSearchFilterFromScopeHierarchy } from './utils/simulatorUtils';
import { NetworkScopeHierarchy } from './types/networkScopeHierarchy';

// TODO: move these type defs to a central location
export const UrlDetailType = {
    NAMESPACE: 'namespace',
    DEPLOYMENT: 'deployment',
    CIDR_BLOCK: 'cidr',
    EXTERNAL_ENTITIES: 'internet',
    EXTERNAL_GROUP: 'external',
} as const;
export type UrlDetailTypeKey = keyof typeof UrlDetailType;
export type UrlDetailTypeValue = (typeof UrlDetailType)[UrlDetailTypeKey];

function getUrlParamsForEntity(type, id): [UrlDetailTypeValue, string] {
    // eslint-disable-next-line @typescript-eslint/no-unsafe-return
    return [UrlDetailType[type], id];
}

export type TopologyComponentProps = {
    model: CustomModel;
    simulation: Simulation;
    selectedNode?: CustomNodeModel;
    simulator: NetworkPolicySimulator;
    setNetworkPolicyModification: SetNetworkPolicyModification;
    edgeState: EdgeState;
    scopeHierarchy: NetworkScopeHierarchy;
};

const TopologyComponent = ({
    model,
    simulation,
    selectedNode,
    simulator,
    setNetworkPolicyModification,
    edgeState,
    scopeHierarchy,
}: TopologyComponentProps) => {
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForNetworkPolicy = hasReadAccess('NetworkPolicy');

    const firstRenderRef = useRef(true);
    const history = useHistory();
    const controller = useVisualizationController();
    const [defaultDeploymentTab, setDefaultDeploymentTab] = useState(deploymentTabs.DETAILS);

    const closeSidebar = useCallback(() => {
        const queryString = clearSimulationQuery(history.location.search);
        history.push(`${networkBasePath}${queryString}`);
    }, [history]);

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
                const newURL = `${networkBasePath}/${newDetailType}/${encodeURIComponent(
                    newDetailId
                )}${queryString}`;
                history.push(newURL);
            } else {
                // otherwise, return to the graph-only state
                history.push(`${networkBasePath}${queryString}`);
            }
        }
    }

    const { deploymentCount } = useFetchDeploymentCount(
        getSearchFilterFromScopeHierarchy(scopeHierarchy)
    );

    function onNodeSelect(id: string) {
        onNodeClick([id]);
    }

    function zoomInCallback() {
        controller.getGraph().scaleBy(4 / 3);
    }

    function zoomOutCallback() {
        controller.getGraph().scaleBy(0.75);
    }

    function fitToScreenCallback() {
        controller.getGraph().fit(80);
    }

    const resetViewCallback = useCallback(() => {
        controller.getGraph().reset();
        controller.getGraph().layout();
    }, [controller]);

    const panNodeIntoView = useCallback(
        (node: CustomNodeModel) => {
            const selectedNodeElement = controller.getNodeById(node.id);
            if (selectedNodeElement) {
                // the offset is to make sure the label also makes it inside the viewport
                controller.getGraph().panIntoView(selectedNodeElement, { offset: 50 });
            }
        },
        [controller]
    );

    useEventListener<SelectionEventListener>(SELECTION_EVENT, (ids) => {
        onNodeClick(ids);
    });

    useEffect(() => {
        // we don't want to reset view on init
        if (!firstRenderRef.current && controller.hasGraph()) {
            resetViewCallback();
        } else {
            firstRenderRef.current = false;
        }
    }, [controller, edgeState, resetViewCallback]);

    useEffect(() => {
        controller.fromModel(model);
        if (selectedNode) {
            panNodeIntoView(selectedNode);
        } else if (history.location.pathname !== networkBasePath && !selectedNode) {
            // if the path does not reflect the selected node state, sync URL to state
            closeSidebar();
        }
    }, [controller, model, selectedNode, history, closeSidebar, panNodeIntoView]);

    const selectedIds = selectedNode ? [selectedNode.id] : [];

    return (
        <TopologyView
            sideBar={
                <TopologySideBar resizable onClose={closeSidebar}>
                    {hasReadAccessForNetworkPolicy &&
                        simulation.isOn &&
                        simulation.type === 'networkPolicy' && (
                            <NetworkPolicySimulatorSidePanel
                                simulator={simulator}
                                setNetworkPolicyModification={setNetworkPolicyModification}
                                scopeHierarchy={scopeHierarchy}
                                scopeDeploymentCount={deploymentCount ?? 0}
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
