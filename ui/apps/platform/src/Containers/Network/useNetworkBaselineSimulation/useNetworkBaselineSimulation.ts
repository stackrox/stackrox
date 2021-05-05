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
};

type ConnectedProps = {
    isBaselineSimulationOn: boolean;
    baselineSimulationOptions: BaselineSimulationOptions;
};

const selectBaselineSimulation = createStructuredSelector<BaselineSimulationState, ConnectedProps>({
    isBaselineSimulationOn: selectors.getIsBaselineSimulationOn,
    baselineSimulationOptions: selectors.getBaselineSimulationOptions,
});

const useNetworkBaselineSimulation = (): BaselineSimulationResult => {
    const dispatch = useDispatch();
    const { isBaselineSimulationOn, baselineSimulationOptions } = useSelector(
        selectBaselineSimulation
    );
    const startBaselineSimulation = (options: BaselineSimulationOptions) => {
        dispatch(actions.startBaselineSimulation(options));
    };
    const stopBaselineSimulation = () => {
        dispatch(actions.stopBaselineSimulation());
    };

    return {
        isBaselineSimulationOn,
        baselineSimulationOptions,
        startBaselineSimulation,
        stopBaselineSimulation,
    };
};

export default useNetworkBaselineSimulation;
