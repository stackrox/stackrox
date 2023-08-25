import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

export const types = {
    FETCH_CLUSTER_INIT_BUNDLES: createFetchingActionTypes(
        'clusterInitBundles/FETCH_CLUSTER_INIT_BUNDLES'
    ),
};

export const actions = {
    fetchClusterInitBundles: createFetchingActions(types.FETCH_CLUSTER_INIT_BUNDLES),
};

const clusterInitBundles = (state = [], action) => {
    if (action.type === types.FETCH_CLUSTER_INIT_BUNDLES.SUCCESS) {
        return isEqual(action.response.items, state) ? state : action.response.items;
    }
    return state;
};

const reducer = combineReducers({
    clusterInitBundles,
});

const getClusterInitBundles = (state) => state.clusterInitBundles;

export const selectors = {
    getClusterInitBundles,
};

export default reducer;
