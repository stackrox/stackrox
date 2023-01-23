/* eslint-disable @typescript-eslint/no-unsafe-return */
import React, { useMemo } from 'react';
import { useHistory, useParams } from 'react-router-dom';
import { action } from 'mobx';
import intersectionWith from 'lodash/intersectionWith';
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
    Visualization,
    VisualizationSurface,
    VisualizationProvider,
    Edge,
    Controller,
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
import { EdgeState } from './components/EdgeStateSelect';
import { getNodeById } from './utils/networkGraphUtils';
import { CustomEdgeModel, CustomModel, CustomNodeModel } from './types/topology.type';
import { createExtraneousEdges } from './utils/modelUtils';
import { Simulation } from './utils/getSimulation';
import LegendContent from './components/LegendContent';

import './Topology.css';
import useNetworkPolicySimulator, {
    ApplyNetworkPolicyModification,
    NetworkPolicySimulator,
    SetNetworkPolicyModification,
} from './hooks/useNetworkPolicySimulator';
import SimulationFrame from './simulation/SimulationFrame';

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
    updateCount: number;
};

export type TopologyComponentProps = {
    model: CustomModel;
    edgeState: EdgeState;
    // simulation: Simulation;
    selectedClusterId: string;
    // simulator: NetworkPolicySimulator;
    // setNetworkPolicyModification: SetNetworkPolicyModification;
    // applyNetworkPolicyModification: ApplyNetworkPolicyModification;
};

function getNodeEdges(selectedNode) {
    return [...selectedNode.getSourceEdges(), ...selectedNode.getTargetEdges()];
}

function setVisibleEdges(edges) {
    edges.forEach((edge) => {
        edge.setVisible(true);
    });
}

// jeff's solution
const setEdgesVisible = action((edges: Edge[], visible: boolean) =>
    edges.forEach((edge) => edge.setVisible(visible))
);

// jeff's solution
const showNodeEdges = (controller: Controller, nodeId: string) => {
    console.log('showNodeEdges');
    if (!nodeId) {
        setEdgesVisible(controller.getGraph().getEdges(), true);
        return;
    }

    setEdgesVisible(controller.getGraph().getEdges(), false);

    const selectedNode = controller.getNodeById(nodeId);
    if (selectedNode?.isGroup()) {
        // set visible edges
        selectedNode
            .getAllNodeChildren()
            .forEach((child) => setEdgesVisible(getNodeEdges(child), true));
    } else if (selectedNode) {
        // set visible edges
        setEdgesVisible(getNodeEdges(selectedNode), true);
    }
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
    edgeState,
    // simulation,
    selectedClusterId,
}: // simulator,
// setNetworkPolicyModification,
// applyNetworkPolicyModification,
TopologyComponentProps) => {
    const history = useHistory();
    const { detailId } = useParams();
    const selectedEntity = detailId && getNodeById(model?.nodes, detailId);
    const controller = useVisualizationController();
    console.log('TopologyComponent');

    function resetGraphToDefault() {
        console.log('TopologyComponent: resetGraphToDefault');
        controller.fromModel(model, true);
    }

    function rerenderGraph() {
        console.log('TopologyComponent: rerenderGraph');
        resetGraphToDefault();
        // setNodes();
        setEdges();
    }

    function showExtraneousNodes() {
        console.log('showExtraneousNodes');
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
        console.log('hideExtraneousNodes');
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
        console.log('setExtraneousEdges');
        const currentModel = controller.toModel() as CustomModel;
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
            const edges: CustomEdgeModel[] = currentModel.edges || [];
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
        console.log('removeExtraneousEdges');
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
        console.log('TopologyComponent: onSelect');
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
        // removeExtraneousEdges();
        console.log('TopologyComponent: setEdges');
        if (detailId) {
            const nodeEdges: CustomEdgeModel[] = [];
            const selectedNode = getNodeById(model.nodes, detailId);
            if (selectedNode?.type === 'group') {
                // need to find the edges that have a source or target that equal to the child id
                model.edges.forEach((edge) => {
                    const isSource = selectedNode?.children?.includes(edge.source);
                    const isTarget = selectedNode?.children?.includes(edge.target);
                    if (isSource || isTarget) {
                        nodeEdges.push({ ...edge, visible: true });
                    }
                });
                console.log(nodeEdges);
                // controller.fromModel({ ...model, edges: nodeEdges });

                // const nodeEdges = intersectionWith(
                //     [model.edges, selectedNode?.children],
                //     (modelEdge, childNode) =>
                //         modelEdge.source === childNode.id || modelEdge.target === childNode.id
                // );

                //    .forEach((child) => {
                //         // set visible edges
                //         model.edges.filter((edge) => edge.)
                //         // setVisibleEdges(getNodeEdges(child));
                //     });
            } else if (selectedNode) {
                // set visible edges
                const nodeEdges = getNodeEdges(selectedNode);
                nodeEdges.forEach((edge) => {
                    const edgeData = edge.getData();
                    edge.setData({ ...edgeData, visible: true });
                });
                // setVisibleEdges(getNodeEdges(selectedNode));
            }
            // // setting extraneous edges
            // if (edgeState === 'extraneous') {
            //     setExtraneousEdges();
            // }

            // // jeff's solution
            // showNodeEdges(controller, detailId);
        }
    }

    React.useEffect(() => {
        console.log('TopologyComponent: useEffect [model, detailId]');
        rerenderGraph();
    }, [model, detailId]);

    useEventListener<SelectionEventListener>(SELECTION_EVENT, (ids) => {
        console.log('TopologyComponent: useEventListener');
        onSelect(ids);
    });

    const selectedIds = selectedEntity ? [selectedEntity.id] : [];

    return (
        <TopologyView
            // sideBar={
            //     <TopologySideBar resizable onClose={closeSidebar}>
            //         {simulation.isOn && simulation.type === 'networkPolicy' && (
            //             <NetworkPolicySimulatorSidePanel
            //                 selectedClusterId={selectedClusterId}
            //                 simulator={simulator}
            //                 setNetworkPolicyModification={setNetworkPolicyModification}
            //                 applyNetworkPolicyModification={applyNetworkPolicyModification}
            //             />
            //         )}
            //         {selectedEntity && selectedEntity?.data?.type === 'NAMESPACE' && (
            //             <NamespaceSideBar
            //                 namespaceId={selectedEntity.id}
            //                 nodes={model?.nodes || []}
            //                 edges={model?.edges || []}
            //             />
            //         )}
            //         {selectedEntity && selectedEntity?.data?.type === 'DEPLOYMENT' && (
            //             <DeploymentSideBar
            //                 deploymentId={selectedEntity.id}
            //                 nodes={model?.nodes || []}
            //                 edges={model?.edges || []}
            //             />
            //         )}
            //         {selectedEntity && selectedEntity?.data?.type === 'EXTERNAL_GROUP' && (
            //             <ExternalGroupSideBar
            //                 id={selectedEntity.id}
            //                 nodes={model?.nodes || []}
            //                 edges={model?.edges || []}
            //             />
            //         )}
            //         {selectedEntity && selectedEntity?.data?.type === 'CIDR_BLOCK' && (
            //             <CidrBlockSideBar
            //                 id={selectedEntity.id}
            //                 nodes={model?.nodes || []}
            //                 edges={model?.edges || []}
            //             />
            //         )}
            //         {selectedEntity && selectedEntity?.data?.type === 'EXTERNAL_ENTITIES' && (
            //             <ExternalEntitiesSideBar
            //                 id={selectedEntity.id}
            //                 nodes={model?.nodes || []}
            //                 edges={model?.edges || []}
            //             />
            //         )}
            //     </TopologySideBar>
            // }
            // sideBarOpen={!!selectedEntity || simulation.isOn}
            // sideBarResizable
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
                            resetGraphToDefault();
                        },
                        resetViewCallback: () => {
                            controller.getGraph().reset();
                            controller.getGraph().layout();
                        },
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

function compareModels(prevProps, nextProps) {
    console.log(
        'NetworkGraph: compareModels prevProps nextProps',
        prevProps.updateCount,
        nextProps.updateCount
    );
    return (
        prevProps.updateCount === nextProps.updateCount &&
        prevProps.simulation.isOn === nextProps.simulation.isOn &&
        prevProps.simulation.type === nextProps.simulation.type
    );
}

const NetworkGraph = React.memo<NetworkGraphProps>(
    ({ model, edgeState, simulation, selectedClusterId, updateCount }) => {
        const controller = useMemo(() => new Visualization(), []);
        controller.registerLayoutFactory(defaultLayoutFactory);
        controller.registerComponentFactory(defaultComponentFactory);
        controller.registerComponentFactory(stylesComponentFactory);

        // const { simulator, setNetworkPolicyModification, applyNetworkPolicyModification } =
        //     useNetworkPolicySimulator({
        //         simulation,
        //         clusterId: selectedClusterId,
        //     });

        const isSimulating =
            simulator.state === 'GENERATED' ||
            simulator.state === 'UNDO' ||
            simulator.state === 'UPLOAD' ||
            (simulation.isOn && simulation.type === 'baseline');

        return (
            // <SimulationFrame simulator={simulator}>
            <VisualizationProvider controller={controller}>
                <TopologyComponent
                    model={model}
                    edgeState={edgeState}
                    // simulation={simulation}
                    selectedClusterId={selectedClusterId}
                    // simulator={simulator}
                    // setNetworkPolicyModification={setNetworkPolicyModification}
                    // applyNetworkPolicyModification={applyNetworkPolicyModification}
                />
            </VisualizationProvider>
            // </SimulationFrame>
        );
    },
    compareModels
);

NetworkGraph.displayName = 'NetworkGraph';

export default NetworkGraph;
