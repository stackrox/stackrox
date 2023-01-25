import { QueryValue } from 'hooks/useURLParameter';

export type SimulationType = 'baseline' | 'networkPolicy';

type SimulationOn = {
    isOn: true;
    type: SimulationType;
};

type SimulationOff = {
    isOn: false;
};

export type Simulation = SimulationOn | SimulationOff;

function getSimulation(simulationQueryValue: QueryValue): Simulation {
    if (
        !simulationQueryValue ||
        (simulationQueryValue !== 'baseline' && simulationQueryValue !== 'networkPolicy')
    ) {
        const simulation: Simulation = {
            isOn: false,
        };
        return simulation;
    }
    const simulation: Simulation = {
        isOn: true,
        type: simulationQueryValue,
    };
    return simulation;
}

export default getSimulation;
