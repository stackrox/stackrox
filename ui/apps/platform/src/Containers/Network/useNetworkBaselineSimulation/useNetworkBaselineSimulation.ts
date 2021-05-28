import { useSelector, useDispatch } from 'react-redux';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';

import {
    actions,
    BaselineSimulationOptions,
    BaselineSimulationState,
} from 'reducers/network/baselineSimulation';

export type BaselineSimulationResult = {
    isBaselineSimulationOn: boolean;
    baselineSimulationOptions: BaselineSimulationOptions;
    startBaselineSimulation: (options: BaselineSimulationOptions) => void;
    stopBaselineSimulation: () => void;
    isUndoOn: boolean;
};

type ConnectedProps = {
    isBaselineSimulationOn: boolean;
    baselineSimulationOptions: BaselineSimulationOptions;
    isUndoOn: boolean;
};

const selectBaselineSimulation = createStructuredSelector<BaselineSimulationState, ConnectedProps>({
    isBaselineSimulationOn: selectors.getIsBaselineSimulationOn,
    baselineSimulationOptions: selectors.getBaselineSimulationOptions,
    isUndoOn: selectors.getIsUndoOn,
});

const useNetworkBaselineSimulation = (): BaselineSimulationResult => {
    const dispatch = useDispatch();
    const { isBaselineSimulationOn, baselineSimulationOptions, isUndoOn } = useSelector(
        selectBaselineSimulation
    );
    const startBaselineSimulation = (options: BaselineSimulationOptions) => {
        dispatch(actions.startBaselineSimulation(options));
    };
    const stopBaselineSimulation = () => {
        dispatch(actions.toggleUndoPreview(false));
        dispatch(actions.stopBaselineSimulation());
    };

    return {
        isBaselineSimulationOn,
        baselineSimulationOptions,
        startBaselineSimulation,
        stopBaselineSimulation,
        isUndoOn,
    };
};

export default useNetworkBaselineSimulation;
