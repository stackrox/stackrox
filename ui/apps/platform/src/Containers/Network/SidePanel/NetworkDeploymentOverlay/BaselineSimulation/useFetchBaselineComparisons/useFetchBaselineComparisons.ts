import { useEffect, useState } from 'react';

import { fetchBaselineComparison } from 'services/NetworkService';
import { FilterState } from 'Containers/Network/networkTypes';
import { SimulatedBaseline } from '../SimulatedNetworkBaselines/baselineSimulationTypes';
import processBaselineComparisons from './processBaselineComparisons';

export type FetchBaselineComparisonsResult = {
    isLoading: boolean;
    simulatedBaselines: SimulatedBaseline[];
    error: string | null;
};

export type UseFetchBaselineComparisons = {
    deploymentId: string;
    filterState: FilterState;
};

const defaultResultState = {
    simulatedBaselines: [],
    error: null,
    isLoading: true,
};

function useFetchBaselineComparisons({
    deploymentId,
    filterState,
}: UseFetchBaselineComparisons): FetchBaselineComparisonsResult {
    const [result, setResult] = useState<FetchBaselineComparisonsResult>(defaultResultState);

    useEffect(() => {
        const baselineComparisonPromise = fetchBaselineComparison({ deploymentId });

        baselineComparisonPromise
            .then((response) => {
                const simulatedBaselines = processBaselineComparisons(response, filterState);
                setResult({
                    simulatedBaselines,
                    error: null,
                    isLoading: false,
                });
            })
            .catch((error) => {
                setResult({
                    simulatedBaselines: [],
                    error,
                    isLoading: false,
                });
            });
    }, [deploymentId, filterState]);

    return result;
}

export default useFetchBaselineComparisons;
