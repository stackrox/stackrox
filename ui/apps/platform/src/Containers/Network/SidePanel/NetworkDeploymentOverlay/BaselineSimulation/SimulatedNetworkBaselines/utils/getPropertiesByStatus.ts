import { SimulatedBaseline, Properties } from '../baselineSimulationTypes';

function getPropertiesByStatus(datum: SimulatedBaseline): Properties {
    if (datum.simulatedStatus === 'ADDED') {
        return datum.peer.added;
    }
    if (datum.simulatedStatus === 'REMOVED') {
        return datum.peer.removed;
    }
    if (datum.simulatedStatus === 'UNMODIFIED') {
        return datum.peer.unmodified;
    }
    throw new Error('Simulated Baseline Status should be either ADDED, REMOVED, or MODIFIED');
}

export default getPropertiesByStatus;
