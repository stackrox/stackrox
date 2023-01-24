import React, { useState, useEffect, useRef } from 'react';
import { useParams } from 'react-router-dom';

import NetworkGraph from './NetworkGraph';
import {
    CustomEdgeModel,
    CustomModel,
    CustomNodeModel,
    DeploymentData,
    DeploymentNodeModel,
    ExtraneousNodeModel,
    NetworkPolicyState,
} from './types/topology.type';
import { EdgeState } from './components/EdgeStateSelect';
import { DisplayOption } from './components/DisplayOptionsSelect';
import { Simulation } from './utils/getSimulation';
import { getNodeById } from './utils/networkGraphUtils';
import { createExtraneousNodes, createExtraneousEdges, graphModel } from './utils/modelUtils';

export type Models = {
    active: CustomModel;
    extraneous: CustomModel;
};

// figure out how to handle namespace edge filtering
function getFilteredEdges(edges: CustomEdgeModel[], detailId: string): CustomEdgeModel[] {
    const filteredEdges: CustomEdgeModel[] = [];
    edges.forEach((edge) => {
        const { source, target } = edge;
        if (source === detailId || target === detailId) {
            filteredEdges.push({ ...edge, visible: true });
        }
    });
    console.log('NetworkGraphContainer: getFilteredEdges', filteredEdges);
    return filteredEdges;
}

function updateEgressFlowsNode(
    node: ExtraneousNodeModel,
    networkPolicyState: NetworkPolicyState
): ExtraneousNodeModel {
    const updatedEgressFlowsNode = { ...node };
    if (networkPolicyState === 'ingress' || networkPolicyState === 'none') {
        // if the node has ingress or no policies from policy graph, show extraneous egress node
        updatedEgressFlowsNode.visible = true;
    } else {
        updatedEgressFlowsNode.visible = false;
    }
    return updatedEgressFlowsNode;
}

function updateIngressFlowsNode(
    node: ExtraneousNodeModel,
    networkPolicyState: NetworkPolicyState
): ExtraneousNodeModel {
    const updatedIngressFlowsNode = { ...node };
    if (networkPolicyState === 'egress' || networkPolicyState === 'none') {
        // if the node has egress or no policies from policy graph, show extraneous ingress node
        updatedIngressFlowsNode.visible = true;
    } else {
        updatedIngressFlowsNode.visible = false;
    }
    return updatedIngressFlowsNode;
}

function getExtraneousNodes(
    nodes: CustomNodeModel[],
    extraneousFlowsNodes: {
        egressFlowsNode: ExtraneousNodeModel;
        ingressFlowsNode: ExtraneousNodeModel;
    },
    selectedNodeData?: DeploymentData
): CustomNodeModel[] {
    const updatedNodes = [...nodes];
    if (selectedNodeData?.type === 'DEPLOYMENT') {
        const { egressFlowsNode, ingressFlowsNode } = extraneousFlowsNodes;
        const { networkPolicyState } = selectedNodeData || {};
        const updatedEgressFlowsNode = updateEgressFlowsNode(egressFlowsNode, networkPolicyState);
        const updatedIngressFlowsNode = updateIngressFlowsNode(
            ingressFlowsNode,
            networkPolicyState
        );
        updatedNodes.push(updatedEgressFlowsNode);
        updatedNodes.push(updatedIngressFlowsNode);
    }
    return updatedNodes;
}

function getExtraneousEdges(
    edges: CustomEdgeModel[],
    extraneousFlowsEdges?: {
        extraneousEgressEdge: CustomEdgeModel;
        extraneousIngressEdge: CustomEdgeModel;
    },
    selectedNodeData?: DeploymentData
): CustomEdgeModel[] {
    const updatedEdges = [...edges];
    // else if there is a selected node, add edges to extraneous flows node(s)
    if (selectedNodeData?.type === 'DEPLOYMENT' && extraneousFlowsEdges) {
        const { extraneousEgressEdge, extraneousIngressEdge } = extraneousFlowsEdges;
        const { networkPolicyState } = selectedNodeData || {};
        if (networkPolicyState === 'ingress') {
            updatedEdges.push(extraneousEgressEdge);
        } else if (networkPolicyState === 'egress') {
            updatedEdges.push(extraneousIngressEdge);
        } else if (networkPolicyState === 'none') {
            updatedEdges.push(extraneousEgressEdge);
            updatedEdges.push(extraneousIngressEdge);
        }
    }
    return updatedEdges;
}

type NetworkGraphContainerProps = {
    models: Models;
    edgeState: EdgeState;
    displayOptions: DisplayOption[];
    simulation: Simulation;
    selectedClusterId: string;
    clusterDeploymentCount: number;
};

function NetworkGraphContainer({
    models,
    edgeState,
    displayOptions,
    simulation,
    selectedClusterId,
    clusterDeploymentCount,
}: NetworkGraphContainerProps) {
    // these are the unfiltered, unmodified data models
    const { active, extraneous } = models;
    // this is the current filtered and/or modified model that is represented in the graph
    const [model, setModel] = useState(active);
    // this is a count to improve performance (we only rerender children when updateCount changes)
    const updateCount = useRef(0);
    // selected node state is stored in the URL
    const { detailId } = useParams();
    const selectedNode = getNodeById(model?.nodes, detailId);
    // extraneous catch-all (egress/ingress flows) nodes to add/remove from extraneous nodes model
    const extraneousFlowsNodes = createExtraneousNodes(clusterDeploymentCount);
    let extraneousFlowsEdges: {
        extraneousEgressEdge: CustomEdgeModel;
        extraneousIngressEdge: CustomEdgeModel;
    };
    if (detailId) {
        extraneousFlowsEdges = createExtraneousEdges(detailId);
    }

    console.log('NetworkGraphContainer', detailId);

    function increaseUpdateCount() {
        updateCount.current += 1;
    }

    useEffect(() => {
        console.log(
            'NetworkGraphContainer: useEffect [displayOptions, detailId, edgeState, active, extraneous]'
        );
        const showPolicyState = !!displayOptions.includes('policyStatusBadge');
        const showExternalState = !!displayOptions.includes('externalBadge');
        const showEdgeLabels = !!displayOptions.includes('edgeLabel');
        let updatedNodes: CustomNodeModel[] = [...active.nodes];
        let updatedEdges: CustomEdgeModel[] = [...active.edges];
        if (edgeState === 'extraneous') {
            updatedNodes = getExtraneousNodes(
                extraneous.nodes,
                extraneousFlowsNodes,
                selectedNode?.data
            );
            updatedEdges = getExtraneousEdges(
                [...extraneous.edges],
                extraneousFlowsEdges,
                selectedNode?.data
            );
        }
        updatedEdges = getFilteredEdges(updatedEdges, detailId);

        // if all display options are true, set back to existing default data model
        if (showPolicyState && showExternalState && showEdgeLabels) {
            increaseUpdateCount();
            setModel({
                ...graphModel,
                nodes: updatedNodes,
                edges: updatedEdges,
            });
        } else {
            // sample test to see if node policy/external state is already showing
            const sampleDeployment = model.nodes.find(
                (node) => node.data.type === 'DEPLOYMENT'
            ) as DeploymentNodeModel;
            const isShowingPolicyState = sampleDeployment.data.showPolicyState;
            const isShowingExternalState = sampleDeployment.data.showExternalState;
            // to improve perf to only perform this if policyStatusBadge OR externalBadge has changed
            if (
                isShowingPolicyState !== showPolicyState ||
                isShowingExternalState !== showExternalState
            ) {
                // update the display options visually for deployment nodes on the graph
                updatedNodes = updatedNodes.map((node) => {
                    const { data } = node;
                    if (data.type === 'DEPLOYMENT') {
                        return {
                            ...node,
                            data: {
                                ...data,
                                showPolicyState,
                                showExternalState,
                            } as DeploymentData,
                        };
                    }
                    return node;
                });
            }

            // sample test to see if the edge labels are already showing in the current model in the graph
            const isShowingEdgeLabels = !!model.edges[0].data.tag;
            // to improve perf to only perform this if showEdgeLabels has changed
            if (isShowingEdgeLabels !== showEdgeLabels) {
                // update the display options visually for edges on the graph
                updatedEdges = updatedEdges.map((edge) => {
                    const { data } = edge;
                    const { properties } = data;
                    return {
                        ...edge,
                        visible: true,
                        data: {
                            ...data,
                            properties,
                            tag: showEdgeLabels ? data.portProtocolLabel : undefined,
                        },
                    };
                });
            }

            const updatedModel: CustomModel = {
                ...model,
                nodes: updatedNodes,
                edges: updatedEdges,
            };
            increaseUpdateCount();
            setModel(updatedModel);
        }
    }, [displayOptions, detailId, edgeState, active, extraneous]);

    return (
        <NetworkGraph
            model={model}
            simulation={simulation}
            selectedClusterId={selectedClusterId || ''}
            updateCount={updateCount.current}
            selectedNode={selectedNode}
        />
    );
}

export default NetworkGraphContainer;
