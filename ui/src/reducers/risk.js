import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

// Action types

export const types = {
    FETCH_DEPLOYMENTS: createFetchingActionTypes('deployments/FETCH_DEPLOYMENTS')
};

// Actions

export const actions = {
    fetchDeployments: createFetchingActions(types.FETCH_DEPLOYMENTS)
};

// Reducers

const deployments = (state = [], action) => {
    if (action.type === types.FETCH_DEPLOYMENTS.SUCCESS) {
        return isEqual(action.response.deployments, state) ? state : action.response.deployments;
    }
    return state;
};

const reducer = combineReducers({
    deployments
});

export default reducer;

// Selectors

const getDeployments = state => state.deployments;

export const selectors = {
    getDeployments
};
