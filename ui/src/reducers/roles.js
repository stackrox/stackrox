import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

export const types = {
    FETCH_ROLES: createFetchingActionTypes('roles/FETCH_ROLES')
};

export const actions = {
    fetchRoles: createFetchingActions(types.FETCH_ROLES)
};

const roles = (state = [], action) => {
    if (action.type === types.FETCH_ROLES.SUCCESS) {
        return isEqual(action.response.roles, state) ? state : action.response.roles;
    }
    return state;
};

const reducer = combineReducers({
    roles
});

const getRoles = state => state.roles;

export const selectors = {
    getRoles
};

export default reducer;
