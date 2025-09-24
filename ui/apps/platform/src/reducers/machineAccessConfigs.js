import isEqual from 'lodash/isEqual';
import { combineReducers } from 'redux';

import { createFetchingActions, createFetchingActionTypes } from 'utils/fetchingReduxRoutines';

export const types = {
    FETCH_MACHINE_ACCESS_CONFIGS: createFetchingActionTypes(
        'machineAccessConfigs/FETCH_MACHINE_ACCESS_CONFIGS'
    ),
    DELETE_MACHINE_ACCESS_CONFIGS: 'machineAccessConfigs/DELETE_MACHINE_ACCESS_CONFIGS',
};

export const actions = {
    fetchMachineAccessConfigs: createFetchingActions(types.FETCH_MACHINE_ACCESS_CONFIGS),
    deleteMachineAccessConfigs: (ids) => ({ type: types.DELETE_MACHINE_ACCESS_CONFIGS, ids }),
};

const machineAccessConfigs = (state = [], action) => {
    if (action.type === types.FETCH_MACHINE_ACCESS_CONFIGS.SUCCESS) {
        return isEqual(action.response.configs, state) ? state : action.response.configs;
    }
    return state;
};

const reducer = combineReducers({
    machineAccessConfigs,
});

const getMachineAccessConfigs = (state) => state.machineAccessConfigs;

export const selectors = {
    getMachineAccessConfigs,
};

export default reducer;
