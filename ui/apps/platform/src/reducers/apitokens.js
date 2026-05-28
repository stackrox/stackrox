import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

export const types = {
    FETCH_API_TOKENS: createFetchingActionTypes('apitokens/FETCH_API_TOKENS'),
};

export const actions = {
    fetchAPITokens: createFetchingActions(types.FETCH_API_TOKENS),
};

const apiTokens = (state = [], action) => {
    if (action.type === types.FETCH_API_TOKENS.SUCCESS) {
        return isEqual(action.response.tokens, state) ? state : action.response.tokens;
    }
    return state;
};

const reducer = combineReducers({
    apiTokens,
});

const getAPITokens = (state) => state.apiTokens;

export const selectors = {
    getAPITokens,
};

export default reducer;
