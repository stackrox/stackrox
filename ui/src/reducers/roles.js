import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

export const types = {
    FETCH_ROLES: createFetchingActionTypes('roles/FETCH_ROLES'),
    SELECTED_ROLE: 'roles/SELECTED_ROLE',
    SAVE_ROLE: 'roles/SAVE_ROLE',
    DELETE_ROLE: 'roles/DELETE_ROLE'
};

export const actions = {
    fetchRoles: createFetchingActions(types.FETCH_ROLES),
    selectRole: role => ({
        type: types.SELECTED_ROLE,
        role
    }),
    saveRole: role => ({
        type: types.SAVE_ROLE,
        role
    }),
    deleteRole: id => ({
        type: types.DELETE_ROLE,
        id
    })
};

const roles = (state = [], action) => {
    if (action.type === types.FETCH_ROLES.SUCCESS) {
        return isEqual(action.response.roles, state) ? state : action.response.roles;
    }
    return state;
};

const selectedRole = (state = null, action) => {
    if (action.type === types.FETCH_ROLES.SUCCESS && !state) {
        if (action.response.roles.length) {
            return action.response.roles[0];
        }
        return state;
    }
    if (action.type === types.SELECTED_ROLE) {
        return isEqual(action.role, state) ? state : action.role;
    }
    return state;
};

const reducer = combineReducers({
    roles,
    selectedRole
});

const getRoles = state => state.roles;
const getSelectedRole = state => state.selectedRole;

export const selectors = {
    getRoles,
    getSelectedRole
};

export default reducer;
