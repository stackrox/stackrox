import { useState } from 'react';

import { BaselineSimulationOptions, BaselineSimulationResult } from './baselineSimulationTypes';

const useBaselineSimulation = (): BaselineSimulationResult => {
    const [simulationMode, setSimulationMode] = useState({
        isBaselineSimulationOn: false,
        baselineSimulationOptions: { excludePortsAndProtocols: false },
    });

    const { isBaselineSimulationOn, baselineSimulationOptions } = simulationMode;
    const startBaselineSimulation = (options: BaselineSimulationOptions) => {
        setSimulationMode({ isBaselineSimulationOn: true, baselineSimulationOptions: options });
    };
    const stopBaselineSimulation = () => {
        setSimulationMode({
            isBaselineSimulationOn: false,
            baselineSimulationOptions: { excludePortsAndProtocols: false },
        });
    };

    return {
        isBaselineSimulationOn,
        baselineSimulationOptions,
        startBaselineSimulation,
        stopBaselineSimulation,
    };
};

export default useBaselineSimulation;
