import { useSelector } from 'react-redux';
import { selectors } from 'reducers';
import { createSelector } from 'reselect';

import { FilterState, SimulatedBaseline } from 'Containers/Network/networkTypes';
import processBaselineComparisons from './processBaselineComparisons';

export type FetchBaselineComparisonsResult = {
    isLoading: boolean;
    simulatedBaselines: SimulatedBaseline[];
    error: Error | null;
};

export type UseFetchBaselineComparisons = {
    deploymentId: string;
    filterState: FilterState;
};

const selectBaselineComparisons = createSelector(
    [selectors.getBaselineComparisons],
    (baselineComparisons) => baselineComparisons
);

function useFetchBaselineComparisons({
    filterState,
}: UseFetchBaselineComparisons): FetchBaselineComparisonsResult {
    const result = useSelector(selectBaselineComparisons);

    const simulatedBaselines = processBaselineComparisons(result.data, filterState);

    return {
        ...result,
        simulatedBaselines,
    };
}

export default useFetchBaselineComparisons;
