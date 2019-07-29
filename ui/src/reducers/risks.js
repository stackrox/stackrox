import { combineReducers } from 'redux';
import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

// Action types
export const types = {
    FETCH_RISK: createFetchingActionTypes('risks/FETCH_RISK')
};

// Actions
export const actions = {
    fetchRisk: createFetchingActions(types.FETCH_RISK)
};

// Reducers
const byDeployment = (state = {}, action) => {
    if (action.type === types.FETCH_RISK.SUCCESS) {
        return action.response;
    }
    return state;
};

const reducer = combineReducers({
    byDeployment
});

export default reducer;

// Selectors
const getRiskByDeployment = state => state.byDeployment;

export const selectors = {
    getRiskByDeployment
};
