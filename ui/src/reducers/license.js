import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';
import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';
import { types as metadataTypes, METADATA_LICENSE_STATUS } from './metadata';

export const LICENSE_STATUS = Object.freeze({
    ...METADATA_LICENSE_STATUS
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
    ACTIVATE_LICENSE: 'license/ACTIVATE_LICENSE'
};

// Actions

export const actions = {
    fetchLicenses: createFetchingActions(types.FETCH_LICENSES),
    setLicenseUploadStatus: (status, message = null) => ({
        type: types.SET_LICENSE_UPLOAD_STATUS,
        data: {
            status,
            message
        }
    }),
    activateLicense: licenseKey => ({
        type: types.ACTIVATE_LICENSE,
        licenseKey
    })
};

// Reducers

const license = (state = null, action) => {
    if (
        action.type === metadataTypes.INITIAL_FETCH_METADATA.SUCCESS ||
        action.type === metadataTypes.POLL_METADATA.SUCCESS
    ) {
        const { licenseStatus } = action.response;
        const newState = {
            ...state,
            status: licenseStatus
        };
        return isEqual(newState, state) ? state : newState;
    }
    if (action.type === types.FETCH_LICENSES.SUCCESS) {
        const { licenses } = action.response;
        if (licenses.length) {
            const data = licenses[0];
            const newState = {
                ...state,
                ...data
            };
            return isEqual(newState, state) ? state : newState;
        }
    }
    return state;
};

const licenseUploadStatus = (state = null, action) => {
    if (action.type === types.SET_LICENSE_UPLOAD_STATUS) {
        return isEqual(action.data, state) ? state : action.data;
    }
    return state;
};

const reducer = combineReducers({
    license,
    licenseUploadStatus
});

export default reducer;

// Selectors

const getLicense = state => state.license;
const getLicenseExpirationDate = state => {
    if (!state.license || !state.license.license || !state.license.license.restrictions)
        return null;
    return state.license.license.restrictions.notValidAfter;
};
const getLicenseStatus = state => {
    if (!state.license) return null;
    return state.license.status;
};
const getLicenseUploadStatus = state => state.licenseUploadStatus;

export const selectors = {
    getLicense,
    getLicenseExpirationDate,
    getLicenseStatus,
    getLicenseUploadStatus
};
