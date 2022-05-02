import { combineReducers } from 'redux';
import { createSelector } from 'reselect';
import isEqual from 'lodash/isEqual';
import pick from 'lodash/pick';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';
import mergeEntitiesById from 'utils/mergeEntitiesById';

// Action types

export const types = {
    FETCH_CLUSTERS: createFetchingActionTypes('clusters/FETCH_CLUSTERS'),
};

// Actions

export const actions = {
    fetchClusters: createFetchingActions(types.FETCH_CLUSTERS),
};

// Reducers

const byId = (state = {}, { type, response }) => {
    if (type === types.FETCH_CLUSTERS.SUCCESS) {
        if (!response.entities.cluster) {
            return {};
        }
        const clustersById = response.entities.cluster;
        const newState = mergeEntitiesById(state, clustersById);
        const onlyExisting = pick(newState, Object.keys(clustersById));
        return isEqual(onlyExisting, state) ? state : onlyExisting;
    }
    return state;
};

const reducer = combineReducers({
    byId,
});

export default reducer;

// Selectors

const getClustersById = (state) => state.byId;
const getClusters = createSelector([getClustersById], (clusters) => Object.values(clusters));

export const selectors = {
    getClustersById,
    getClusters,
};
