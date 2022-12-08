import { SearchFilter } from 'types/search';

type SimulationOn = {
    isOn: true;
    type: 'baseline' | 'network policy';
};

type SimulationOff = {
    isOn: false;
};

export type Simulation = SimulationOn | SimulationOff;

function getSimulation(searchFilter: SearchFilter): Simulation {
    const simulationType = searchFilter.Simulation;
    if (!simulationType || (simulationType !== 'baseline' && simulationType !== 'network policy')) {
        const simulation: Simulation = {
            isOn: false,
        };
        return simulation;
    }
    const simulation: Simulation = {
        isOn: true,
        type: simulationType,
    };
    return simulation;
}

export default getSimulation;
