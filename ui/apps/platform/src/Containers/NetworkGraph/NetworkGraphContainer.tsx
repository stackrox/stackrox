import React from 'react';
import { useParams } from 'react-router-dom';

import relatedEntitySVG from 'images/network-graph/related-entity.svg';
import filteredEntitySVG from 'images/network-graph/filtered-entity.svg';

import NetworkGraph from './NetworkGraph';
import {
    CIDRBlockData,
    CustomEdgeModel,
    CustomModel,
    CustomNodeModel,
    DeploymentData,
    ExtraneousNodeModel,
    NamespaceData,
    NetworkPolicyState,
} from './types/topology.type';
import { Simulation } from './utils/getSimulation';
import { getNodeById } from './utils/networkGraphUtils';
import { EdgeState } from './components/EdgeStateSelect';
import { DisplayOption } from './components/DisplayOptionsSelect';
import {
    createExtraneousNodes,
    createExtraneousEdges,
    graphModel,
    getConnectedNodeIds,
} from './utils/modelUtils';
import {
    cidrBlockBadgeColor,
    cidrBlockBadgeText,
    deploymentBadgeColor,
    deploymentBadgeText,
    namespaceBadgeColor,
    namespaceBadgeText,
} from './common/NetworkGraphIcons';
import { NetworkScopeHierarchy } from './types/networkScopeHierarchy';

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
    showExternalState: boolean,
    showSelectionIndicators: boolean,
    showObjectTypeLabels: boolean
): CustomNodeModel[] {
    return nodes.map((node) => {
        const { data } = node;
        if (data.type === 'DEPLOYMENT') {
            let deploymentData: DeploymentData = {
                ...data,
                showPolicyState,
                showExternalState,
            };
            if (showObjectTypeLabels) {
                deploymentData = {
                    ...deploymentData,
                    badge: deploymentBadgeText,
                    badgeColor: deploymentBadgeColor,
                    badgeTextColor: '#FFFFFF',
                    badgeBorderColor: '#FFFFFF',
                };
            }
            return {
                ...node,
                data: deploymentData,
            };
        }
        if (data.type === 'NAMESPACE') {
            let namespaceData: NamespaceData = {
                ...data,
            };
            if (showObjectTypeLabels) {
                namespaceData = {
                    ...data,
                    badge: namespaceBadgeText,
                    badgeColor: namespaceBadgeColor,
                    badgeTextColor: '#FFFFFF',
                    badgeBorderColor: '#FFFFFF',
                };
            }
            if (showSelectionIndicators) {
                namespaceData = {
                    ...namespaceData,
                    labelIconClass: data.isFilteredNamespace ? filteredEntitySVG : relatedEntitySVG,
                };
            }
            return {
                ...node,
                data: namespaceData,
            };
        }
        if (data.type === 'CIDR_BLOCK') {
            let cidrData: CIDRBlockData = {
                ...data,
            };
            if (showObjectTypeLabels) {
                cidrData = {
                    ...data,
                    badge: cidrBlockBadgeText,
                    badgeColor: cidrBlockBadgeColor,
                    badgeTextColor: '#FFFFFF',
                    badgeBorderColor: '#FFFFFF',
                };
            }
            return {
                ...node,
                data: cidrData,
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

// This function modifies the nodes to add another data attribute to distinguish faded out nodes from normal ones
function fadeOutUnconnectedNodes(
    nodes: CustomNodeModel[],
    edges: CustomEdgeModel[],
    selectedNodeId: string | undefined
): CustomNodeModel[] {
    const selectedNode = getNodeById(nodes, selectedNodeId);
    // if nothing is selected we don't want to fade anything out
    if (!selectedNodeId || !selectedNode) {
        return nodes;
    }
    let emphasizedNodeIds: string[] = [];
    if (selectedNode.children) {
        // if the selected node is a group, we want to emphasize all connected nodes of the children
        emphasizedNodeIds =
            selectedNode.children.reduce((acc, currId) => {
                const connectedNodeIds = getConnectedNodeIds(edges, currId);
                return [...acc, ...connectedNodeIds];
            }, [] as string[]) || [];
        // we include the child nodes so that they aren't faded out
        emphasizedNodeIds = [...emphasizedNodeIds, ...selectedNode.children];
    } else {
        // if the selected node is not a group, we want to emphasize all connected nodes of the selected node
        emphasizedNodeIds = getConnectedNodeIds(edges, selectedNodeId);
        // we include the selected node so that it isn't faded out
        emphasizedNodeIds = [...emphasizedNodeIds, selectedNodeId];
    }
    const modifiedNodes: CustomNodeModel[] = nodes.map((node) => {
        const { data } = node;
        let isFadedOut = false;
        if (node.children) {
            isFadedOut = !node.children.some((childNodeId) =>
                emphasizedNodeIds.includes(childNodeId)
            );
        } else {
            isFadedOut = !emphasizedNodeIds.includes(node.id);
        }
        return {
            ...node,
            data: {
                ...data,
                isFadedOut,
            },
        } as CustomNodeModel;
    });
    return modifiedNodes;
}

type NetworkGraphContainerProps = {
    models: Models;
    edgeState: EdgeState;
    displayOptions: DisplayOption[];
    simulation: Simulation;
    clusterDeploymentCount: number;
    scopeHierarchy: NetworkScopeHierarchy;
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
    clusterDeploymentCount,
    scopeHierarchy,
}: NetworkGraphContainerProps) {
    // these are the unfiltered, unmodified data models
    const { activeModel, extraneousModel } = models;

    // 1. edgeState base data model setting ----------------------------------------------
    // this is the unfiltered, unmodified base data model based on edgeState
    const baseModel = edgeState === 'active' ? activeModel : extraneousModel;

    // 2. selectedNode/edgeState data model filtering ------------------------------------
    // selected node state is stored in the URL
    const { detailId: encodedDetailId } = useParams();
    const detailId = decodeURIComponent(encodedDetailId);
    const selectedNode = getNodeById(baseModel?.nodes, detailId);
    // extraneous catch-all in/egress flows nodes to add/remove from extraneous nodes model
    const extraneousNodes = createExtraneousNodes(clusterDeploymentCount);
    // this is the current filtered model that has not been modified yet
    let filteredNodes: CustomNodeModel[] = [...baseModel.nodes];
    let filteredEdges: CustomEdgeModel[] = [...baseModel.edges];
    // if edgeState is extraneous && there is a selectedNode, add in/egress flows nodes/edges
    if (edgeState === 'inactive' && selectedNode?.data.type === 'DEPLOYMENT') {
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
    const showSelectionIndicators = !!displayOptions.includes('selectionIndicator');
    const showObjectTypeLabels = !!displayOptions.includes('objectTypeLabel');
    // modified filtered nodes/edges based on selected displayOptions
    let modifiedNodes = filteredNodes;
    let modifiedEdges = filteredEdges;
    // update the display options visually for deployment nodes on the graph
    modifiedNodes = getDisplayNodes(
        filteredNodes,
        showPolicyState,
        showExternalState,
        showSelectionIndicators,
        showObjectTypeLabels
    );
    // update the display options visually for edges on the graph
    modifiedEdges = getDisplayEdges(filteredEdges, showEdgeLabels);

    // fade out some nodes based on which nodes are connected to the selected node
    modifiedNodes = fadeOutUnconnectedNodes(modifiedNodes, modifiedEdges, selectedNode?.id);

    // this is the resulting model that is passed to the NetworkGraph to render as-is
    const updatedModel: CustomModel = {
        graph: graphModel,
        nodes: modifiedNodes,
        edges: modifiedEdges,
    };

    return (
        <NetworkGraph
            model={updatedModel}
            simulation={simulation}
            selectedNode={selectedNode}
            edgeState={edgeState}
            scopeHierarchy={scopeHierarchy}
        />
    );
}

export default NetworkGraphContainer;
