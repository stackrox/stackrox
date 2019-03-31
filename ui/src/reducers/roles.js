import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

export const ACCESS_LEVEL = Object.freeze({
    READ_WRITE_ACCESS: 'READ_WRITE_ACCESS',
    READ_ACCESS: 'READ_ACCESS',
    NO_ACCESS: 'NO_ACCESS'
});

export const types = {
    FETCH_USER_ROLE_PERMISSIONS: createFetchingActionTypes('roles/FETCH_USER_ROLE_PERMISSIONS'),
    FETCH_ROLES: createFetchingActionTypes('roles/FETCH_ROLES'),
    SELECTED_ROLE: 'roles/SELECTED_ROLE',
    SAVE_ROLE: 'roles/SAVE_ROLE',
    DELETE_ROLE: 'roles/DELETE_ROLE'
};

export const actions = {
    fetchUserRolePermissions: createFetchingActions(types.FETCH_USER_ROLE_PERMISSIONS),
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
    if (action.type === types.SELECTED_ROLE && action.role) {
        return isEqual(action.role, state) ? state : action.role;
    }
    return state;
};

const userRolePermissions = (state = null, action) => {
    if (action.type === types.FETCH_USER_ROLE_PERMISSIONS.SUCCESS) {
        return isEqual(action.response, state) ? state : action.response;
    }
    return state;
};

const reducer = combineReducers({
    roles,
    selectedRole,
    userRolePermissions
});

const getRoles = state => state.roles;
const getSelectedRole = state => state.selectedRole;

const getAccessForPermission = (state, permission) => {
    if (!state.userRolePermissions) return true;
    const { globalAccess, resourceToAccess } = state.userRolePermissions;
    const access = !resourceToAccess ? globalAccess : resourceToAccess[permission];
    return access;
};

const hasReadPermission = state => permission => {
    const access = getAccessForPermission(state, permission);
    return access === ACCESS_LEVEL.READ_WRITE_ACCESS || access === ACCESS_LEVEL.READ_ACCESS;
};
const hasReadWritePermission = state => permission => {
    const access = getAccessForPermission(state, permission);
    return access === ACCESS_LEVEL.READ_WRITE_ACCESS;
};

export const selectors = {
    getRoles,
    getSelectedRole,
    hasReadPermission,
    hasReadWritePermission
};

export default reducer;
