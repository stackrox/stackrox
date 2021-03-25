import React, { createContext, ReactElement, ReactNode } from 'react';

import useBaselineSimulation from './useBaselineSimulation';
import { BaselineSimulationResult } from './baselineSimulationTypes';

const BaselineSimulationContext = createContext<BaselineSimulationResult | undefined>(undefined);

type BaselineSimulationProviderProps = { children: ReactNode };

export function BaselineSimulationProvider({
    children,
}: BaselineSimulationProviderProps): ReactElement {
    const baselineSimulationResult = useBaselineSimulation();
    return (
        <BaselineSimulationContext.Provider value={baselineSimulationResult}>
            {children}
        </BaselineSimulationContext.Provider>
    );
}

export function useNetworkBaselineSimulation(): BaselineSimulationResult {
    const context = React.useContext(BaselineSimulationContext);
    if (context === undefined) {
        throw new Error(
            'useNetworkBaselineSimulation must be used within a BaselineSimulationProvider'
        );
    }
    return context;
}
