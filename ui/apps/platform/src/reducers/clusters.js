import { combineReducers } from 'redux';
import { createSelector } from 'reselect';
import isEqual from 'lodash/isEqual';
import pick from 'lodash/pick';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';
import {
    types as searchTypes,
    getActions as getSearchActions,
    reducers as searchReducers,
    getSelectors as getSearchSelectors,
} from 'reducers/pageSearch';
import mergeEntitiesById from 'utils/mergeEntitiesById';

// Action types

export const types = {
    FETCH_CLUSTERS: createFetchingActionTypes('clusters/FETCH_CLUSTERS'),
    ...searchTypes('clusters'),
};

// Actions

export const actions = {
    fetchClusters: createFetchingActions(types.FETCH_CLUSTERS),
    ...getSearchActions('clusters'),
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
    ...searchReducers('clusters'),
});

export default reducer;

// Selectors

const getClustersById = (state) => state.byId;
const getClusters = createSelector([getClustersById], (clusters) => Object.values(clusters));

export const selectors = {
    getClustersById,
    getClusters,
    ...getSearchSelectors('clusters'),
};
