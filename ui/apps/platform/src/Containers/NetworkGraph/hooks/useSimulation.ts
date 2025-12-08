import useURLParameter from 'hooks/useURLParameter';
import getSimulation from '../utils/getSimulation';
import type { Simulation, SimulationType } from '../utils/getSimulation';

export type UseSimulationResult = {
    simulation: Simulation;
    setSimulation: (type: SimulationType) => void;
};

function useSimulation() {
    const [simulationQueryValue, setSimulationQueryValue] = useURLParameter(
        'simulation',
        undefined
    );
    const simulation = getSimulation(simulationQueryValue);
    return {
        simulation,
        setSimulation: (type: SimulationType) => {
            setSimulationQueryValue(type);
        },
    };
}

export default useSimulation;
