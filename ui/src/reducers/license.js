import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';
import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';
import { types as metadataTypes } from './metadata';

export const LICENSE_STATUS = Object.freeze({
    UNKNOWN: 'UNKNOWN',
    VALID: 'VALID',
    REVOKED: 'REVOKED',
    NOT_YET_VALID: 'NOT_YET_VALID',
    EXPIRED: 'EXPIRED',
    OTHER: 'OTHER'
});

export const LICENSE_UPLOAD_STATUS = Object.freeze({
    VERIFYING: 'VERIFYING',
    VALID: 'VALID',
    INVALID: 'INVALID',
    EXPIRED: 'EXPIRED'
});

// Action types

export const types = {
    FETCH_LICENSES: createFetchingActionTypes('license/FETCH_LICENSES'),
    SET_LICENSE_UPLOAD_STATUS: 'license/SET_LICENSE_UPLOAD_STATUS',
    ACTIVATE_LICENSE: 'license/ACTIVATE_LICENSE',
    SHOW_LICENSE_REMINDER: 'license/SHOW_LICENSE_REMINDER',
    DISMISS_LICENSE_REMINDER: 'license/DISMISS_LICENSE_REMINDER'
};

// Actions

export const actions = {
    fetchLicenses: createFetchingActions(types.FETCH_LICENSES),
    setLicenseUploadStatus: status => ({
        type: types.SET_LICENSE_UPLOAD_STATUS,
        status
    }),
    activateLicense: licenseKey => ({
        type: types.ACTIVATE_LICENSE,
        licenseKey
    }),
    showLicenseReminder: () => ({
        type: types.SHOW_LICENSE_REMINDER
    }),
    dismissLicenseReminder: () => ({
        type: types.DISMISS_LICENSE_REMINDER
    })
};

// Reducers

const license = (state = null, action) => {
    if (action.type === metadataTypes.INITIAL_FETCH_METADATA.SUCCESS) {
        const licenseStatus = action.response.licenseStatus || LICENSE_STATUS.VALID;
        const response = {
            status: licenseStatus
        };
        return isEqual(response, state) ? state : response;
    }
    if (action.type === types.FETCH_LICENSES.SUCCESS) {
        const { licenses } = action.response;
        if (licenses.length)
            return isEqual(action.response.licenses[0], state)
                ? state
                : action.response.licenses[0];
    }
    return state;
};

const showLicenseReminder = (state = false, action) => {
    if (action.type === types.SHOW_LICENSE_REMINDER) {
        return true;
    }
    if (action.type === types.DISMISS_LICENSE_REMINDER) {
        return false;
    }
    return state;
};

const licenseUploadStatus = (state = null, action) => {
    if (action.type === types.SET_LICENSE_UPLOAD_STATUS) {
        return isEqual(action.status, state) ? state : action.status;
    }
    return state;
};

const reducer = combineReducers({
    license,
    showLicenseReminder,
    licenseUploadStatus
});

export default reducer;

// Selectors

const getLicense = state => state.license;
const getLicenseExpirationDate = state => {
    try {
        return state.license.license.restrictions.notValidAfter;
    } catch (error) {
        return null;
    }
};
const getLicenseStatus = state => {
    try {
        return state.license.status;
    } catch (error) {
        return state.license;
    }
};
const shouldShowLicenseReminder = state => state.showLicenseReminder;
const getLicenseUploadStatus = state => state.licenseUploadStatus;

export const selectors = {
    getLicense,
    getLicenseExpirationDate,
    getLicenseStatus,
    shouldShowLicenseReminder,
    getLicenseUploadStatus
};
