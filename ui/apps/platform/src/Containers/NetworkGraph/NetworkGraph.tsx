import React, { useMemo } from 'react';
import { Visualization, VisualizationProvider } from '@patternfly/react-topology';

import stylesComponentFactory from './components/stylesComponentFactory';
import defaultLayoutFactory from './layouts/defaultLayoutFactory';
import defaultComponentFactory from './components/defaultComponentFactory';
import type { CustomModel, CustomNodeModel } from './types/topology.type';
import type { Simulation } from './utils/getSimulation';

import './Topology.css';
import type {
    NetworkPolicySimulator,
    SetNetworkPolicyModification,
} from './hooks/useNetworkPolicySimulator';
import SimulationFrame from './simulation/SimulationFrame';
import TopologyComponent from './TopologyComponent';
import type { EdgeState } from './components/EdgeStateSelect';
import type { NetworkScopeHierarchy } from './types/networkScopeHierarchy';

export type NetworkGraphProps = {
    isReadyForVisualization: boolean;
    model: CustomModel;
    simulation: Simulation;
    selectedNode?: CustomNodeModel;
    edgeState: EdgeState;
    simulator: NetworkPolicySimulator;
    setNetworkPolicyModification: SetNetworkPolicyModification;
    scopeHierarchy: NetworkScopeHierarchy;
    isSimulating: boolean;
};
function NetworkGraph({
    isReadyForVisualization,
    model,
    simulation,
    selectedNode,
    edgeState,
    simulator,
    setNetworkPolicyModification,
    scopeHierarchy,
    isSimulating,
}: NetworkGraphProps) {
    const controller = useMemo(() => {
        const newController = new Visualization();
        newController.registerLayoutFactory(defaultLayoutFactory);
        newController.registerComponentFactory(defaultComponentFactory);
        newController.registerComponentFactory(stylesComponentFactory);
        return newController;
    }, []);

    return (
        <SimulationFrame isSimulating={isSimulating}>
            <VisualizationProvider controller={controller}>
                <TopologyComponent
                    isReadyForVisualization={isReadyForVisualization}
                    model={model}
                    simulation={simulation}
                    simulator={simulator}
                    selectedNode={selectedNode}
                    setNetworkPolicyModification={setNetworkPolicyModification}
                    edgeState={edgeState}
                    scopeHierarchy={scopeHierarchy}
                />
            </VisualizationProvider>
        </SimulationFrame>
    );
}

NetworkGraph.displayName = 'NetworkGraph';

export default NetworkGraph;
