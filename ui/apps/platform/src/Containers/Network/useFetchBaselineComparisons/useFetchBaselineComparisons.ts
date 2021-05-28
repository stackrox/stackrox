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

const selectUndoComparisons = createSelector(
    [selectors.getUndoComparisons],
    (undoComparisons) => undoComparisons
);

const selectNetworkFilterMode = createSelector(
    [selectors.getNetworkGraphFilterMode],
    (filterMode) => filterMode as FilterState
);

const selectIsUndoOn = createSelector([selectors.getIsUndoOn], (isUndoOn) => isUndoOn);

function useFetchBaselineComparisons(): FetchBaselineComparisonsResult {
    const baselineComparisons = useSelector(selectBaselineComparisons);
    const undoComparisons = useSelector(selectUndoComparisons);
    const filterState = useSelector(selectNetworkFilterMode);
    const isUndoOn = useSelector(selectIsUndoOn);

    const comparisonsToUse = isUndoOn ? undoComparisons : baselineComparisons;
    const simulatedBaselines = processBaselineComparisons(comparisonsToUse.data, filterState);

    return {
        ...comparisonsToUse,
        simulatedBaselines,
    };
}

export default useFetchBaselineComparisons;
