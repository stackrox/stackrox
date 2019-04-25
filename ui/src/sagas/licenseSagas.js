import Raven from 'raven-js';
import store from 'store';

import { all, take, fork, put, takeLatest, select, call } from 'redux-saga/effects';
import { push } from 'react-router-redux';
import { licenseStartUpPath, licensePath, authResponsePrefix } from 'routePaths';
import { takeEveryLocation } from 'utils/sagaEffects';
import { selectors } from 'reducers';
import { actions, types, LICENSE_STATUS, LICENSE_UPLOAD_STATUS } from 'reducers/license';
import { types as locationActionTypes } from 'reducers/routes';
import { fetchLicenses, addLicense } from 'services/LicenseService';
import { actions as notificationActions } from 'reducers/notifications';
import { types as metadataTypes } from 'reducers/metadata';
import { pollUntilCentralRestarts } from 'sagas/metadataSagas';

export const storeRequestedLocation = location => store.set('license_requested_location', location);
export const getAndClearRequestedLocation = () => {
    const location = store.get('license_requested_location');
    store.remove('license_requested_location');
    return location;
};

export function* getLicenses() {
    try {
        const result = yield call(fetchLicenses);
        yield put(actions.fetchLicenses.success(result.response));
    } catch (error) {
        // do nothing
        Raven.captureException(error);
    }
}

function* addNewLicense(data) {
    let errorMessage = 'There was an error uploading the license';
    const successMessage = 'The license was successfully added';

    try {
        yield put(actions.setLicenseUploadStatus(LICENSE_UPLOAD_STATUS.VERIFYING));
        const response = yield call(addLicense, data);
        if (
            response.accepted &&
            response.license.active &&
            response.license.status === LICENSE_STATUS.VALID
        ) {
            yield call(pollUntilCentralRestarts);
            yield fork(getLicenses);
            yield put(actions.setLicenseUploadStatus(LICENSE_UPLOAD_STATUS.VALID, successMessage));
            return;
        }

        errorMessage = response.accepted
            ? 'The license was accepted, but is not being used at the moment'
            : 'The license was rejected';

        if (response.license.statusReason) {
            errorMessage += `: ${response.license.statusReason}`;
        }
    } catch (error) {
        if (error.response) {
            errorMessage += `: ${error.response.data.error}`;
        } else {
            Raven.captureException(error);
        }
    }
    yield put(actions.setLicenseUploadStatus(LICENSE_UPLOAD_STATUS.INVALID, errorMessage));
    yield put(notificationActions.addNotification(errorMessage));
    yield put(notificationActions.removeOldestNotification());
}

function* checkLicenseStatus(location) {
    const status = yield select(selectors.getLicenseStatus);
    if (!status) {
        // if there's no license status, we need to fetch licenses and check
        storeRequestedLocation(location);
        yield fork(getLicenses);
    } else if (status === LICENSE_STATUS.VALID) {
        // if the license is valid, redirect them back to their requested route
        const storedLocation = getAndClearRequestedLocation();
        if (!storedLocation) return;
        yield fork(getLicenses);
        if (storedLocation.pathname === licenseStartUpPath) {
            yield put(push('/'));
        } else {
            yield put(push(storedLocation || '/'));
        }
    } else {
        // if the license is expired, the user should be directed to the license start up page
        storeRequestedLocation(location);
        yield put(push('/license'));
    }
}

function* activateLicense(data) {
    yield fork(addNewLicense, data);
}

function* clearLicenseUploadStatus() {
    yield put(actions.setLicenseUploadStatus(null));
}

function* watchLicenseActivation() {
    yield takeLatest(types.ACTIVATE_LICENSE, activateLicense);
}

function* watchFetchLicense() {
    yield takeLatest(types.FETCH_LICENSES.SUCCESS, checkLicenseStatus);
}

function* watchMetadataLicenseStatus() {
    yield takeLatest(metadataTypes.INITIAL_FETCH_METADATA.SUCCESS, checkLicenseStatus);
}

export default function* license() {
    yield fork(watchFetchLicense);
    yield fork(watchMetadataLicenseStatus);
    const action = yield take(locationActionTypes.LOCATION_CHANGE);
    const { payload: location } = action;
    if (location.pathname && !location.pathname.startsWith(authResponsePrefix)) {
        yield fork(checkLicenseStatus, location);
    }
    yield all([
        takeEveryLocation(licensePath, clearLicenseUploadStatus),
        fork(watchLicenseActivation)
    ]);
}
