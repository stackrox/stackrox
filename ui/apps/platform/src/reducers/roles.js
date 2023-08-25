import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';
import { replacedResourceMapping } from '../constants/accessControl';

export const ACCESS_LEVEL = Object.freeze({
    READ_WRITE_ACCESS: 'READ_WRITE_ACCESS',
    READ_ACCESS: 'READ_ACCESS',
    NO_ACCESS: 'NO_ACCESS',
});

export const types = {
    FETCH_USER_ROLE_PERMISSIONS: createFetchingActionTypes('roles/FETCH_USER_ROLE_PERMISSIONS'),
    FETCH_ROLES: createFetchingActionTypes('roles/FETCH_ROLES'),
    SELECTED_ROLE: 'roles/SELECTED_ROLE',
    SAVE_ROLE: 'roles/SAVE_ROLE',
    DELETE_ROLE: 'roles/DELETE_ROLE',
};

export const actions = {
    fetchUserRolePermissions: createFetchingActions(types.FETCH_USER_ROLE_PERMISSIONS),
    fetchRoles: createFetchingActions(types.FETCH_ROLES),
    selectRole: (role) => ({
        type: types.SELECTED_ROLE,
        role,
    }),
    saveRole: (role) => ({
        type: types.SAVE_ROLE,
        role,
    }),
    deleteRole: (id) => ({
        type: types.DELETE_ROLE,
        id,
    }),
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

const error = (state = null, action) => {
    switch (action.type) {
        case types.FETCH_USER_ROLE_PERMISSIONS.REQUEST:
        case types.FETCH_USER_ROLE_PERMISSIONS.SUCCESS:
            return null;

        case types.FETCH_USER_ROLE_PERMISSIONS.FAILURE:
            return action.error;

        default:
            return state;
    }
};

const isLoading = (state = true, action) => {
    // Initialize true for edge case before authSagas call fetchUserRolePermissions action.
    switch (action.type) {
        case types.FETCH_USER_ROLE_PERMISSIONS.REQUEST:
            return true;

        case types.FETCH_USER_ROLE_PERMISSIONS.FAILURE:
        case types.FETCH_USER_ROLE_PERMISSIONS.SUCCESS:
            return false;

        default:
            return state;
    }
};

const reducer = combineReducers({
    roles,
    selectedRole,
    userRolePermissions,
    error,
    isLoading,
});

const getRoles = (state) => state.roles;
const getSelectedRole = (state) => state.selectedRole;
const getUserRolePermissions = (state) => state.userRolePermissions;
const getUserRolePermissionsError = (state) => state.error;
const getIsLoadingUserRolePermissions = (state) => state.isLoading;

/*
 * Given resource string (for example, "Integration") and role or permissionSet object,
 * return access level (for example, "READ_ACCESS").
 */
const getAccessForPermission = (resource, userRolePermissionsArg) => {
    return userRolePermissionsArg?.resourceToAccess?.[resource] ?? ACCESS_LEVEL.NO_ACCESS;
};

export const getHasReadPermission = (resource, userRolePermissionsArg) => {
    const access = getAccessForPermission(resource, userRolePermissionsArg);
    if (access === ACCESS_LEVEL.READ_WRITE_ACCESS || access === ACCESS_LEVEL.READ_ACCESS) {
        return true;
    }
    // If the given resource doesn't yield the required access, try with the replacing resource (if there is any).
    if (replacedResourceMapping.has(resource)) {
        const replacingResourceAccess = getAccessForPermission(
            replacedResourceMapping.get(resource),
            userRolePermissionsArg
        );
        return (
            replacingResourceAccess === ACCESS_LEVEL.READ_WRITE_ACCESS ||
            replacingResourceAccess === ACCESS_LEVEL.READ_ACCESS
        );
    }
    // Return false if neither the resource nor the replacing resource have the correct access.
    return false;
};

export const getHasReadWritePermission = (resource, userRolePermissionsArg) => {
    const access = getAccessForPermission(resource, userRolePermissionsArg);
    if (access === ACCESS_LEVEL.READ_WRITE_ACCESS) {
        return true;
    }
    // If the given resource doesn't yield the required access, try with the replacing resource (if there is any).
    if (replacedResourceMapping.has(resource)) {
        const replacingResourceAccess = getAccessForPermission(
            replacedResourceMapping.get(resource),
            userRolePermissionsArg
        );
        return replacingResourceAccess === ACCESS_LEVEL.READ_WRITE_ACCESS;
    }
    // Return false if neither the resource nor the replacing resource have the correct access.
    return false;
};

export const selectors = {
    getRoles,
    getSelectedRole,
    getUserRolePermissions,
    getUserRolePermissionsError,
    getIsLoadingUserRolePermissions,
};

export default reducer;
