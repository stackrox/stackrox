import React, { useState, useEffect, useRef } from 'react';
import { useParams } from 'react-router-dom';

import NetworkGraph from './NetworkGraph';
import {
    CustomEdgeModel,
    CustomModel,
    CustomNodeModel,
    DeploymentData,
    DeploymentNodeModel,
} from './types/topology.type';
import { EdgeState } from './components/EdgeStateSelect';
import { DisplayOption } from './components/DisplayOptionsSelect';
import { Simulation } from './utils/getSimulation';
import { getNodeById } from './utils/networkGraphUtils';

function getFilteredEdges(edges: CustomEdgeModel[], detailId: string): CustomEdgeModel[] {
    const filteredEdges: CustomEdgeModel[] = [];
    // edges.forEach((edge) => {
    //     const { source, target } = edge;
    //     if (source === detailId || target === detailId) {
    //         filteredEdges.push({ ...edge, visible: true });
    //     }
    // });
    return filteredEdges;
}

// function getFilteredNodes(nodes: CustomNodeModel[], selectedNode: CustomNodeModel, edgeState: EdgeState): CustomNodeModel[] {
//     const { data } = selectedNode || {};
//     const updatedNodes = nodes;
//     if (edgeState === 'extraneous') {
//         if (data?.type === 'DEPLOYMENT') {
//             const { networkPolicyState } = data || {};
//             const extraneousIngressNode = nodes.find(({ id }) => id === 'extraneous-ingress');
//             const extraneousEgressNode = nodes.find(({ id }) => id === 'extraneous-egress');
//             if (networkPolicyState === 'ingress') {
//                 // if the node has ingress policies from policy graph, show extraneous egress node
//                 extraneousEgressNode?.setVisible(true);
//             } else if (networkPolicyState === 'egress') {
//                 // if the node has egress policies from policy graph, show extraneous ingress node
//                 extraneousIngressNode?.setVisible(true);
//             } else if (networkPolicyState === 'none') {
//                 // if the node has no policies, show both extraneous ingress and egress nodes
//                 extraneousEgressNode?.setVisible(true);
//                 extraneousIngressNode?.setVisible(true);
//             }
//         }
//     }
//     return updatedNodes;
// }

type NetworkGraphContainerProps = {
    models: {
        active: CustomModel;
        extraneous: CustomModel;
    };
    edgeState: EdgeState;
    displayOptions: DisplayOption[];
    simulation: Simulation;
    selectedClusterId: string;
};

function NetworkGraphContainer({
    models,
    edgeState,
    displayOptions,
    simulation,
    selectedClusterId,
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
        const unfilteredModel = edgeState === 'active' ? active : extraneous;
        let updatedNodes: CustomNodeModel[] = unfilteredModel.nodes;
        let updatedEdges: CustomEdgeModel[] = getFilteredEdges(unfilteredModel.edges, detailId);

        // if all display options are true, set back to existing default data model
        if (showPolicyState && showExternalState && showEdgeLabels) {
            increaseUpdateCount();
            setModel({
                ...unfilteredModel,
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
                updatedNodes = unfilteredModel.nodes.map((node) => {
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
            edgeState={edgeState}
            simulation={simulation}
            selectedClusterId={selectedClusterId || ''}
            updateCount={updateCount.current}
            selectedNode={selectedNode}
        />
    );
}

export default NetworkGraphContainer;
