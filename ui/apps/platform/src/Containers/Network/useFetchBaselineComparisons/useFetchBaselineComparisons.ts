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

const selectBaselineComparisons = createSelector(
    [selectors.getBaselineComparisons],
    (baselineComparisons) => baselineComparisons
);

const selectNetworkFilterMode = createSelector(
    [selectors.getNetworkGraphFilterMode],
    (filterMode) => filterMode as FilterState
);

function useFetchBaselineComparisons(): FetchBaselineComparisonsResult {
    const result = useSelector(selectBaselineComparisons);
    const filterState = useSelector(selectNetworkFilterMode);

    const simulatedBaselines = processBaselineComparisons(result.data, filterState);

    return {
        ...result,
        simulatedBaselines,
    };
}

export default useFetchBaselineComparisons;
