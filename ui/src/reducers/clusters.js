import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

// Action types

export const types = {
    FETCH_CLUSTERS: createFetchingActionTypes('clusters/FETCH_CLUSTERS')
};

// Actions

export const actions = {
    fetchClusters: createFetchingActions(types.FETCH_CLUSTERS)
};

// Reducers

const clusters = (state = [], action) => {
    if (action.type === types.FETCH_CLUSTERS.SUCCESS) {
        return isEqual(action.response.clusters, state) ? state : action.response.clusters;
    }
    return state;
};

const reducer = combineReducers({
    clusters
});

export default reducer;

// Selectors

const getClusters = state => state.clusters;

export const selectors = {
    getClusters
};
