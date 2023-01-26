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

export type NetworkGraphProps = {
    model: CustomModel;
    simulation: Simulation;
    selectedNode?: CustomNodeModel;
    selectedClusterId: string;
    edgeState: EdgeState;
};
function NetworkGraph({
    model,
    simulation,
    selectedClusterId,
    selectedNode,
    edgeState,
}: NetworkGraphProps) {
    const controller = useMemo(() => new Visualization(), []);
    controller.registerLayoutFactory(defaultLayoutFactory);
    controller.registerComponentFactory(defaultComponentFactory);
    controller.registerComponentFactory(stylesComponentFactory);

    const { simulator, setNetworkPolicyModification, applyNetworkPolicyModification } =
        useNetworkPolicySimulator({
            simulation,
            clusterId: selectedClusterId,
        });

    const isSimulating =
        simulator.state === 'GENERATED' ||
        simulator.state === 'UNDO' ||
        simulator.state === 'UPLOAD' ||
        (simulation.isOn && simulation.type === 'baseline');

    console.log('NetworkGraph');

    return (
        <SimulationFrame isSimulating={isSimulating}>
            <VisualizationProvider controller={controller}>
                <TopologyComponent
                    model={model}
                    simulation={simulation}
                    selectedClusterId={selectedClusterId}
                    simulator={simulator}
                    selectedNode={selectedNode}
                    setNetworkPolicyModification={setNetworkPolicyModification}
                    applyNetworkPolicyModification={applyNetworkPolicyModification}
                    edgeState={edgeState}
                />
            </VisualizationProvider>
        </SimulationFrame>
    );
}

NetworkGraph.displayName = 'NetworkGraph';

export default NetworkGraph;
