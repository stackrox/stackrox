import { combineReducers } from 'redux';
import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

import isEqual from 'lodash/isEqual';

export const types = {
    FETCH_USERS_ATTRIBUTES: createFetchingActionTypes('groups/FETCH_USERS_ATTRIBUTES')
};

export const actions = {
    fetchUsersAttributes: createFetchingActions(types.FETCH_USERS_ATTRIBUTES)
};

const usersAttributes = (state = [], action) => {
    if (action.type === types.FETCH_USERS_ATTRIBUTES.SUCCESS) {
        return isEqual(action.response.usersAttributes, state)
            ? state
            : action.response.usersAttributes;
    }
    return state;
};
const reducer = combineReducers({
    usersAttributes
});

const getUsersAttributes = state => state.usersAttributes;

export const selectors = {
    getUsersAttributes
};

export default reducer;
