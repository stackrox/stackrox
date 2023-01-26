import React from 'react';
import { useParams } from 'react-router-dom';

import NetworkGraph from './NetworkGraph';
import {
    CustomEdgeModel,
    CustomModel,
    CustomNodeModel,
    DeploymentData,
    ExtraneousNodeModel,
    NetworkPolicyState,
} from './types/topology.type';
import { EdgeState } from './components/EdgeStateSelect';
import { DisplayOption } from './components/DisplayOptionsSelect';
import { Simulation } from './utils/getSimulation';
import { getNodeById } from './utils/networkGraphUtils';
import { createExtraneousNodes, createExtraneousEdges, graphModel } from './utils/modelUtils';

export type Models = {
    activeModel: CustomModel;
    extraneousModel: CustomModel;
};

function getFilteredEdges(
    edges: CustomEdgeModel[],
    selectedNode: CustomNodeModel
): CustomEdgeModel[] {
    const filteredEdges: CustomEdgeModel[] = [];
    if (selectedNode.data && selectedNode.data.type !== 'EXTRANEOUS') {
        edges.forEach((edge) => {
            const { source, target } = edge;
            const { type } = selectedNode.data;
            if (type === 'NAMESPACE' || type === 'EXTERNAL_GROUP') {
                // if a namespace is selected, add children's node edges
                const { children } = selectedNode;
                if (children?.includes(source) || children?.includes(target)) {
                    filteredEdges.push({ ...edge, visible: true });
                }
                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                // @ts-ignore
            } else if (source === selectedNode.data?.id || target === selectedNode.data?.id) {
                filteredEdges.push({ ...edge, visible: true });
            }
        });
    }
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

// returns the in/egress flows nodes based on the selected node's policy state
function getExtraneousNodes(
    extraneousFlowsNodes: {
        egressFlowsNode: ExtraneousNodeModel;
        ingressFlowsNode: ExtraneousNodeModel;
    },
    selectedNodeData: DeploymentData
): CustomNodeModel[] {
    const extraneousNodes: CustomNodeModel[] = [];
    if (selectedNodeData.type === 'DEPLOYMENT') {
        const { egressFlowsNode, ingressFlowsNode } = extraneousFlowsNodes;
        const { networkPolicyState } = selectedNodeData;
        const updatedEgressFlowsNode = updateEgressFlowsNode(egressFlowsNode, networkPolicyState);
        const updatedIngressFlowsNode = updateIngressFlowsNode(
            ingressFlowsNode,
            networkPolicyState
        );
        extraneousNodes.push(updatedEgressFlowsNode);
        extraneousNodes.push(updatedIngressFlowsNode);
    }
    return extraneousNodes;
}

// returns edges to the in/egress flows nodes based on the selected node's policy state
function getExtraneousEdges(selectedNodeData: DeploymentData): CustomEdgeModel[] {
    const updatedEdges: CustomEdgeModel[] = [];
    const { id, networkPolicyState } = selectedNodeData;
    const { extraneousEgressEdge, extraneousIngressEdge } = createExtraneousEdges(id);
    if (networkPolicyState === 'ingress') {
        updatedEdges.push(extraneousEgressEdge);
    } else if (networkPolicyState === 'egress') {
        updatedEdges.push(extraneousIngressEdge);
    } else if (networkPolicyState === 'none') {
        updatedEdges.push(extraneousEgressEdge);
        updatedEdges.push(extraneousIngressEdge);
    }
    return updatedEdges;
}

// returns modified nodes based on display options for nodes
function getDisplayNodes(
    nodes: CustomNodeModel[],
    showPolicyState: boolean,
    showExternalState: boolean
): CustomNodeModel[] {
    return nodes.map((node) => {
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

// returns modified edges based on display options for edges
function getDisplayEdges(edges: CustomEdgeModel[], showEdgeLabels: boolean): CustomEdgeModel[] {
    return edges.map((edge) => {
        const { data } = edge;
        return {
            ...edge,
            visible: true,
            data: {
                ...data,
                tag: showEdgeLabels ? data.portProtocolLabel : undefined,
            },
        };
    });
}

type NetworkGraphContainerProps = {
    models: Models;
    edgeState: EdgeState;
    displayOptions: DisplayOption[];
    simulation: Simulation;
    selectedClusterId: string;
    clusterDeploymentCount: number;
};

// the order of model modification is as follows:
// 1. edgeState determines whether to use the activeModel or the extraneousModel
// 2. from the selected edgeState model (baseModel), we add/remove related edges
//    based on the selectedNode and the edgeState (extraneousFlows nodes/edges)
// 3. from the filtered model, we modify the individual properties of each node/edge
//    based on displayOptions
//
// 1 (edgeState) -> 2 (selectedNode/edgeState) -> 3 (displayOptions)
function NetworkGraphContainer({
    models,
    edgeState,
    displayOptions,
    simulation,
    selectedClusterId,
    clusterDeploymentCount,
}: NetworkGraphContainerProps) {
    // these are the unfiltered, unmodified data models
    const { activeModel, extraneousModel } = models;

    // 1. edgeState base data model setting ----------------------------------------------
    // this is the unfiltered, unmodified base data model based on edgeState
    const baseModel = edgeState === 'active' ? activeModel : extraneousModel;

    // 2. selectedNode/edgeState data model filtering ------------------------------------
    // selected node state is stored in the URL
    const { detailId } = useParams();
    const selectedNode = getNodeById(baseModel?.nodes, detailId);
    // extraneous catch-all in/egress flows nodes to add/remove from extraneous nodes model
    const extraneousNodes = createExtraneousNodes(clusterDeploymentCount);
    // this is the current filtered model that has not been modified yet
    let filteredNodes: CustomNodeModel[] = [...baseModel.nodes];
    let filteredEdges: CustomEdgeModel[] = [...baseModel.edges];
    // if edgeState is extraneous && there is a selectedNode, add in/egress flows nodes/edges
    if (edgeState === 'extraneous' && selectedNode?.data.type === 'DEPLOYMENT') {
        const extraneousFlowsNodes = getExtraneousNodes(extraneousNodes, selectedNode.data);
        filteredNodes = [...extraneousModel.nodes, ...extraneousFlowsNodes];
        const extraneousFlowsEdges = getExtraneousEdges(selectedNode.data);
        filteredEdges = [...extraneousModel.edges, ...extraneousFlowsEdges];
    }
    // filtering nodes/edges based on selection, edges will be [] by default
    filteredEdges = selectedNode ? getFilteredEdges(filteredEdges, selectedNode) : [];

    // 3. displayOptions data model modifying --------------------------------------------
    const showPolicyState = !!displayOptions.includes('policyStatusBadge');
    const showExternalState = !!displayOptions.includes('externalBadge');
    const showEdgeLabels = !!displayOptions.includes('edgeLabel');
    // modified filtered nodes/edges based on selected displayOptions
    let modifiedNodes = filteredNodes;
    let modifiedEdges = filteredEdges;
    // update the display options visually for deployment nodes on the graph
    modifiedNodes = getDisplayNodes(filteredNodes, showPolicyState, showExternalState);
    // update the display options visually for edges on the graph
    modifiedEdges = getDisplayEdges(filteredEdges, showEdgeLabels);

    // this is the resulting model that is passed to the NetworkGraph to render as-is
    const updatedModel: CustomModel = {
        graph: graphModel,
        nodes: modifiedNodes,
        edges: modifiedEdges,
    };

    console.log('NetworkGraphContainer');

    return (
        <NetworkGraph
            model={updatedModel}
            simulation={simulation}
            selectedClusterId={selectedClusterId || ''}
            selectedNode={selectedNode}
            edgeState={edgeState}
        />
    );
}

export default NetworkGraphContainer;
