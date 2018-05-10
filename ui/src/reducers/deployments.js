import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';
import {
    types as searchTypes,
    getActions as getSearchActions,
    reducers as searchReducers,
    getSelectors as getSearchSelectors
} from 'reducers/pageSearch';

// Action types

export const types = {
    FETCH_DEPLOYMENTS: createFetchingActionTypes('deployments/FETCH_DEPLOYMENTS'),
    ...searchTypes('deployments')
};

// Actions

export const actions = {
    fetchDeployments: createFetchingActions(types.FETCH_DEPLOYMENTS),
    ...getSearchActions('deployments')
};

// Reducers

const deployments = (state = [], action) => {
    if (action.type === types.FETCH_DEPLOYMENTS.SUCCESS) {
        return isEqual(action.response.deployments, state) ? state : action.response.deployments;
    }
    return state;
};

const reducer = combineReducers({
    deployments,
    ...searchReducers('deployments')
});

export default reducer;

// Selectors

const getDeployments = state => state.deployments;

export const selectors = {
    getDeployments,
    ...getSearchSelectors('deployments')
};
