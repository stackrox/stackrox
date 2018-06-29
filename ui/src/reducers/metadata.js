import { combineReducers } from 'redux';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

// Action types

export const types = {
    FETCH_METADATA: createFetchingActionTypes('metadata/FETCH_METADATA')
};

// Actions

export const actions = {
    fetchMetadata: createFetchingActions(types.FETCH_METADATA)
};

// Reducers

const metadata = (state = null, action) => {
    if (action.type === types.FETCH_METADATA.SUCCESS) {
        return action.response;
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
