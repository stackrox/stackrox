import React, { useState, useEffect, useRef } from 'react';

import NetworkGraph from './NetworkGraph';
import {
    CustomEdgeModel,
    CustomModel,
    CustomNodeModel,
    DeploymentData,
} from './types/topology.type';
import { EdgeState } from './components/EdgeStateSelect';
import { DisplayOption } from './components/DisplayOptionsSelect';
import { Simulation } from './utils/getSimulation';

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
    const { active, extraneous } = models;
    const [model, setModel] = useState(active);
    const updateCount = useRef(0);

    function increaseUpdateCount() {
        updateCount.current += 1;
    }

    useEffect(() => {
        const showPolicyState = !!displayOptions.includes('policyStatusBadge');
        const showExternalState = !!displayOptions.includes('externalBadge');
        const showEdgeLabels = !!displayOptions.includes('edgeLabel');
        let updatedNodes: CustomNodeModel[] = model.nodes;
        let updatedEdges: CustomEdgeModel[] = model.edges;

        // if all display options are true, set back to existing default data model
        if (showPolicyState && showExternalState && showEdgeLabels) {
            increaseUpdateCount();
            setModel(edgeState === 'active' ? active : extraneous);
        } else {
            // this is to update the display options visually for deployment nodes on the graph
            if (model.nodes?.length) {
                // need to improve perf to only perform this if policyStatusBadge OR externalBadge has changed
                updatedNodes = model.nodes.map((node) => {
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

            if (model.edges?.length) {
                // need to improve perf to only perform this if edgeLabel has changed
                updatedEdges = model.edges.map((edge) => {
                    const { data } = edge;
                    const { properties } = data;
                    return {
                        ...edge,
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
    }, [displayOptions]);

    return (
        <NetworkGraph
            model={model}
            edgeState={edgeState}
            simulation={simulation}
            selectedClusterId={selectedClusterId || ''}
            updateCount={updateCount.current}
        />
    );
}

export default NetworkGraphContainer;
