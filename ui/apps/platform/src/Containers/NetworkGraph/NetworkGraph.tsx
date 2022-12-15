/* eslint-disable @typescript-eslint/no-unsafe-return */
import React from 'react';
import { useHistory, useParams } from 'react-router-dom';
import {
    SELECTION_EVENT,
    TopologySideBar,
    TopologyView,
    createTopologyControlButtons,
    defaultControlButtonsOptions,
    TopologyControlBar,
    useVisualizationController,
    Visualization,
    VisualizationSurface,
    VisualizationProvider,
    EdgeModel,
} from '@patternfly/react-topology';

import { networkBasePathPF } from 'routePaths';
import { getQueryObject, getQueryString } from 'utils/queryStringUtils';
import stylesComponentFactory from './components/stylesComponentFactory';
import defaultLayoutFactory from './layouts/defaultLayoutFactory';
import defaultComponentFactory from './components/defaultComponentFactory';
import DeploymentSideBar from './deployment/DeploymentSideBar';
import NamespaceSideBar from './namespace/NamespaceSideBar';
import CidrBlockSideBar from './cidr/CidrBlockSideBar';
import ExternalEntitiesSideBar from './externalEntities/ExternalEntitiesSideBar';
import ExternalGroupSideBar from './external/ExternalGroupSideBar';
import NetworkPolicySimulatorSidePanel from './simulation/NetworkPolicySimulatorSidePanel';
import { EdgeState } from './EdgeStateSelect';
import { getNodeById } from './utils/networkGraphUtils';
import { CustomModel, CustomNodeModel } from './types/topology.type';
import { createExtraneousEdges } from './utils/modelUtils';
import { Simulation } from './utils/getSimulation';

import './Topology.css';

// TODO: move these type defs to a central location
export const UrlDetailType = {
    NAMESPACE: 'namespace',
    DEPLOYMENT: 'deployment',
    CIDR_BLOCK: 'cidr',
    EXTERNAL_ENTITIES: 'internet',
    EXTERNAL: 'external',
} as const;
export type UrlDetailTypeKey = keyof typeof UrlDetailType;
export type UrlDetailTypeValue = typeof UrlDetailType[UrlDetailTypeKey];

function getUrlParamsForEntity(selectedEntity: CustomNodeModel): [UrlDetailTypeValue, string] {
    const detailType = UrlDetailType[selectedEntity.data.type];
    const detailId = selectedEntity.id;

    return [detailType, detailId];
}

export type NetworkGraphProps = {
    model: CustomModel;
    edgeState: EdgeState;
    simulation: Simulation;
    selectedClusterId: string;
};

export type TopologyComponentProps = {
    model: CustomModel;
    edgeState: EdgeState;
    simulation: Simulation;
    selectedClusterId: string;
};

function getNodeEdges(selectedNode) {
    const egressEdges = selectedNode.getSourceEdges();
    const ingressEdges = selectedNode.getTargetEdges();
    return [...egressEdges, ...ingressEdges];
}

function setVisibleEdges(edges) {
    edges.forEach((edge) => {
        edge.setVisible(true);
    });
}

// @TODO: Consider a better approach to managing the side panel related state (simulation + URL path for entities)
function clearSimulationQuery(search: string): string {
    const modifiedSearchFilter = getQueryObject(search);
    // eslint-disable-next-line @typescript-eslint/dot-notation
    if (modifiedSearchFilter.s && modifiedSearchFilter['s']['Simulation']) {
        // eslint-disable-next-line @typescript-eslint/dot-notation
        delete modifiedSearchFilter['s']['Simulation'];
    }
    const queryString = getQueryString(modifiedSearchFilter);
    return queryString;
}

const TopologyComponent = ({
    model,
    edgeState,
    simulation,
    selectedClusterId,
}: TopologyComponentProps) => {
    const history = useHistory();
    const { detailId } = useParams();
    const selectedEntity = detailId && getNodeById(model?.nodes, detailId);
    const controller = useVisualizationController();

    // to prevent error where graph hasn't initialized yet
    if (controller.hasGraph()) {
        rerenderGraph();
    }

    function rerenderGraph() {
        setNodes();
        setEdges();
    }

    function showExtraneousNodes() {
        // else if there is a selected node, create a node to collect extraneous flows
        const selectedNode = controller.getNodeById(detailId);
        // TODO: figure out if/how to support namespaces
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore TS2339: Property 'data' does not exist on type 'Node<NodeModel, any> | {}'.
        const { data } = selectedNode || {};
        if (data?.type === 'DEPLOYMENT') {
            const { networkPolicyState } = data || {};
            const extraneousIngressNode = controller.getElementById('extraneous-ingress');
            const extraneousEgressNode = controller.getElementById('extraneous-egress');
            if (networkPolicyState === 'ingress') {
                // if the node has ingress policies from policy graph, show extraneous egress node
                extraneousEgressNode?.setVisible(true);
            } else if (networkPolicyState === 'egress') {
                // if the node has egress policies from policy graph, show extraneous ingress node
                extraneousIngressNode?.setVisible(true);
            } else if (networkPolicyState === 'none') {
                // if the node has no policies, show both extraneous ingress and egress nodes
                extraneousEgressNode?.setVisible(true);
                extraneousIngressNode?.setVisible(true);
            }
        }
    }

    function hideExtraneousNodes() {
        // if there is no selected node, check if extraneous nodes exist and remove them
        const extraneousIngressNode = controller.getElementById('extraneous-ingress');
        if (extraneousIngressNode) {
            extraneousIngressNode.setVisible(false);
        }
        const extraneousEgressNode = controller.getElementById('extraneous-egress');
        if (extraneousEgressNode) {
            extraneousEgressNode.setVisible(false);
        }
    }

    function setExtraneousEdges() {
        const currentModel = controller.toModel();
        const extraneousIngressNode = controller.getElementById('extraneous-ingress');
        const extraneousEgressNode = controller.getElementById('extraneous-egress');
        const { extraneousEgressEdge, extraneousIngressEdge } = createExtraneousEdges(detailId);
        const selectedNode = controller.getNodeById(detailId);
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore TS2339: Property 'data' does not exist on type 'Node<NodeModel, any> | {}'.
        const { data } = selectedNode || {};
        // else if there is a selected node, create a node to collect extraneous flows
        // TODO: figure out if/how to support namespaces
        if (data?.type === 'DEPLOYMENT') {
            const { networkPolicyState } = data || {};
            const edges: EdgeModel[] = currentModel.edges || [];
            if (networkPolicyState === 'ingress' && extraneousEgressNode) {
                edges.push(extraneousEgressEdge);
            } else if (networkPolicyState === 'egress' && extraneousIngressNode) {
                edges.push(extraneousIngressEdge);
            } else if (
                networkPolicyState === 'none' &&
                extraneousEgressNode &&
                extraneousIngressNode
            ) {
                edges.push(extraneousEgressEdge);
                edges.push(extraneousIngressEdge);
            }
            currentModel.edges = edges;
            controller.fromModel(currentModel);
        }
    }

    function removeExtraneousEdges() {
        // if there is no selected node, check if extraneous edges exist and remove them
        const extraneousIngressEdge = controller.getElementById('extraneous-ingress-edge');
        if (extraneousIngressEdge) {
            controller.removeElement(extraneousIngressEdge);
        }
        const extraneousEgressEdge = controller.getElementById('extraneous-egress-edge');
        if (extraneousEgressEdge) {
            controller.removeElement(extraneousEgressEdge);
        }
    }

    function closeSidebar() {
        const queryString = clearSimulationQuery(history.location.search);
        history.push(`${networkBasePathPF}${queryString}`);
    }

    function onSelect(ids: string[]) {
        const newSelectedId = ids?.[0] || '';
        const newSelectedEntity = getNodeById(model?.nodes, newSelectedId);
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        if (newSelectedEntity) {
            const [newDetailType, newDetailId] = getUrlParamsForEntity(newSelectedEntity);
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

    function setNodes() {
        hideExtraneousNodes();
        if (edgeState === 'extraneous' && detailId) {
            showExtraneousNodes();
        }
    }

    // TODO: figure out how to add/show edges more performantly/smoothly
    function setEdges() {
        removeExtraneousEdges();
        controller
            .getGraph()
            .getEdges()
            .forEach((edge) => {
                edge.setVisible(false);
            });

        if (detailId) {
            const selectedNode = controller.getNodeById(detailId);
            if (selectedNode?.isGroup()) {
                selectedNode.getAllNodeChildren().forEach((child) => {
                    // set visible edges
                    setVisibleEdges(getNodeEdges(child));
                });
            } else if (selectedNode) {
                // set visible edges
                setVisibleEdges(getNodeEdges(selectedNode));
            }

            // setting extraneous edges
            if (edgeState === 'extraneous') {
                setExtraneousEdges();
            }
        }
    }

    React.useEffect(() => {
        controller.fromModel(model, false);
        controller.addEventListener(SELECTION_EVENT, onSelect);

        rerenderGraph();

        return () => {
            controller.removeEventListener(SELECTION_EVENT, onSelect);
        };
    }, [controller, model]);

    const selectedIds = selectedEntity ? [selectedEntity.id] : [];

    return (
        <TopologyView
            sideBar={
                <TopologySideBar resizable onClose={closeSidebar}>
                    {simulation.isOn && simulation.type === 'networkPolicy' && (
                        <NetworkPolicySimulatorSidePanel selectedClusterId={selectedClusterId} />
                    )}
                    {selectedEntity && selectedEntity?.data?.type === 'NAMESPACE' && (
                        <NamespaceSideBar
                            namespaceId={selectedEntity.id}
                            nodes={model?.nodes || []}
                            edges={model?.edges || []}
                        />
                    )}
                    {selectedEntity && selectedEntity?.data?.type === 'DEPLOYMENT' && (
                        <DeploymentSideBar
                            deploymentId={selectedEntity.id}
                            nodes={model?.nodes || []}
                            edges={model?.edges || []}
                        />
                    )}
                    {selectedEntity && selectedEntity?.data?.type === 'EXTERNAL' && (
                        <ExternalGroupSideBar
                            id={selectedEntity.id}
                            nodes={model?.nodes || []}
                            edges={model?.edges || []}
                        />
                    )}
                    {selectedEntity && selectedEntity?.data?.type === 'CIDR_BLOCK' && (
                        <CidrBlockSideBar
                            id={selectedEntity.id}
                            nodes={model?.nodes || []}
                            edges={model?.edges || []}
                        />
                    )}
                    {selectedEntity && selectedEntity?.data?.type === 'EXTERNAL_ENTITIES' && (
                        <ExternalEntitiesSideBar
                            id={selectedEntity.id}
                            nodes={model?.nodes || []}
                            edges={model?.edges || []}
                        />
                    )}
                </TopologySideBar>
            }
            sideBarOpen={!!selectedEntity || simulation.isOn}
            sideBarResizable
            controlBar={
                <TopologyControlBar
                    controlButtons={createTopologyControlButtons({
                        ...defaultControlButtonsOptions,
                        zoomInCallback: () => {
                            controller.getGraph().scaleBy(4 / 3);
                        },
                        zoomOutCallback: () => {
                            controller.getGraph().scaleBy(0.75);
                        },
                        fitToScreenCallback: () => {
                            controller.getGraph().fit(80);
                        },
                        resetViewCallback: () => {
                            controller.getGraph().reset();
                            controller.getGraph().layout();
                        },
                        legendCallback: () => {
                            // console.log('hi');
                        },
                    })}
                />
            }
        >
            <VisualizationSurface state={{ selectedIds }} />
        </TopologyView>
    );
};

const NetworkGraph = React.memo<NetworkGraphProps>(
    ({ model, edgeState, simulation, selectedClusterId }) => {
        const controller = new Visualization();
        controller.registerLayoutFactory(defaultLayoutFactory);
        controller.registerComponentFactory(defaultComponentFactory);
        controller.registerComponentFactory(stylesComponentFactory);

        return (
            <div className="pf-ri__topology-demo">
                <VisualizationProvider controller={controller}>
                    <TopologyComponent
                        model={model}
                        edgeState={edgeState}
                        simulation={simulation}
                        selectedClusterId={selectedClusterId}
                    />
                </VisualizationProvider>
            </div>
        );
    }
);

NetworkGraph.displayName = 'NetworkGraph';

export default NetworkGraph;
