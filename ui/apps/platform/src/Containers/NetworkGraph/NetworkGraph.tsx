import React, { useMemo } from 'react';
import { Visualization, VisualizationProvider } from '@patternfly/react-topology';

import stylesComponentFactory from './components/stylesComponentFactory';
import defaultLayoutFactory from './layouts/defaultLayoutFactory';
import defaultComponentFactory from './components/defaultComponentFactory';
import { CustomModel, CustomNodeModel } from './types/topology.type';
import { Simulation } from './utils/getSimulation';

import './Topology.css';
import useNetworkPolicySimulator from './hooks/useNetworkPolicySimulator';
import SimulationFrame from './simulation/SimulationFrame';
import TopologyComponent from './TopologyComponent';
import { EdgeState } from './components/EdgeStateSelect';
import { NetworkScopeHierarchy } from './types/networkScopeHierarchy';

export type NetworkGraphProps = {
    model: CustomModel;
    simulation: Simulation;
    selectedNode?: CustomNodeModel;
    edgeState: EdgeState;
    scopeHierarchy: NetworkScopeHierarchy;
};
function NetworkGraph({
    model,
    simulation,
    selectedNode,
    edgeState,
    scopeHierarchy,
}: NetworkGraphProps) {
    const controller = useMemo(() => {
        const newController = new Visualization();
        newController.registerLayoutFactory(defaultLayoutFactory);
        newController.registerComponentFactory(defaultComponentFactory);
        newController.registerComponentFactory(stylesComponentFactory);
        return newController;
    }, []);
    const { simulator, setNetworkPolicyModification } = useNetworkPolicySimulator({
        simulation,
        scopeHierarchy,
    });

    const isSimulating =
        simulator.state === 'GENERATED' ||
        simulator.state === 'UNDO' ||
        simulator.state === 'UPLOAD' ||
        (simulation.isOn && simulation.type === 'baseline');

    return (
        <SimulationFrame isSimulating={isSimulating}>
            <VisualizationProvider controller={controller}>
                <TopologyComponent
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
