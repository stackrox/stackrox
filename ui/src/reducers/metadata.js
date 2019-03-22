import { combineReducers } from 'redux';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

// Action types

export const types = {
    INITIAL_FETCH_METADATA: createFetchingActionTypes('metadata/INITIAL_FETCH_METADATA'),
    POLL_METADATA: createFetchingActionTypes('metadata/POLL_METADATA')
};

// Actions

export const actions = {
    initialFetchMetadata: createFetchingActions(types.INITIAL_FETCH_METADATA),
    pollMetadata: createFetchingActions(types.POLL_METADATA)
};

// Reducers

const metadata = (state = {}, action) => {
    if (action.type === types.INITIAL_FETCH_METADATA.SUCCESS) {
        return { ...action.response, stale: false };
    }
    if (action.type === types.POLL_METADATA.SUCCESS) {
        if (action.response.version !== state.version) {
            return Object.assign({}, state, { stale: true });
        }
        if (state.stale) {
            return Object.assign({}, state, { stale: false });
        }
        return state;
    }
    return state;
};

const reducer = combineReducers({
    metadata
});

export default reducer;

// Selectors

const getMetadata = state => state.metadata;

export const selectors = {
    getMetadata
};
