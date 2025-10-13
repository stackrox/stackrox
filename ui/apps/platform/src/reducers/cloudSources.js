import isEqual from 'lodash/isEqual';

import { combineReducers } from 'redux';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

export const types = {
    FETCH_CLOUD_SOURCES: createFetchingActionTypes('cloudSources/FETCH_CLOUD_SOURCES'),
    DELETE_CLOUD_SOURCES: 'cloudSources/DELETE_CLOUD_SOURCES',
};

export const actions = {
    fetchCloudSources: createFetchingActions(types.FETCH_CLOUD_SOURCES),
    deleteCloudSources: (ids) => ({ type: types.DELETE_CLOUD_SOURCES, ids }),
};

const cloudSources = (state = [], action) => {
    if (action.type === types.FETCH_CLOUD_SOURCES.SUCCESS) {
        return isEqual(action.response.cloudSources, state) ? state : action.response.cloudSources;
    }
    return state;
};

const reducer = combineReducers({
    cloudSources,
});

const getCloudSources = (state) => state.cloudSources;

export const selectors = {
    getCloudSources,
};

export default reducer;
