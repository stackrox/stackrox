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
    FETCH_ENVIRONMENT_GRAPH: createFetchingActionTypes('environment/FETCH_ENVIRONMENT_GRAPH'),
    ...searchTypes('environment')
};

// Actions

export const actions = {
    fetchEnvironmentGraph: createFetchingActions(types.FETCH_ENVIRONMENT_GRAPH),
    ...getSearchActions('environment')
};

// Reducers

const environmentGraph = (state = { nodes: [], edges: [] }, action) => {
    if (action.type === types.FETCH_ENVIRONMENT_GRAPH.SUCCESS) {
        return isEqual(action.response, state) ? state : action.response;
    }
    return state;
};

const reducer = combineReducers({
    environmentGraph,
    ...searchReducers('environment')
});

// Selectors

const getEnvironmentGraph = state => state.environmentGraph;

export const selectors = {
    getEnvironmentGraph,
    ...getSearchSelectors('environment')
};

export default reducer;
