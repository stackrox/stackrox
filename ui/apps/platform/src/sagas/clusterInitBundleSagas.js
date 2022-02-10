import { all, take, takeLatest, call, fork, put, select } from 'redux-saga/effects';

import { integrationsPath } from 'routePaths';
import * as service from 'services/ClustersService';
import { actions, types, clusterInitBundleFormId } from 'reducers/clusterInitBundles';
import { actions as roleActions } from 'reducers/roles';
import { actions as notificationActions } from 'reducers/notifications';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';
import { getFormValues } from 'redux-form';

function* getClusterInitBundles() {
    try {
        const result = yield call(service.fetchClusterInitBundles);
        yield put(actions.fetchClusterInitBundles.success(result.response));
    } catch (error) {
        yield put(actions.fetchClusterInitBundles.failure(error));
    }
}

function* generateClusterInitBundle() {
    try {
        const formData = yield select(getFormValues(clusterInitBundleFormId));
        const result = yield call(service.generateClusterInitBundle, formData);
        yield put(actions.generateClusterInitBundle.success(result.response));
    } catch (error) {
        if (error.response) {
            yield put(notificationActions.addNotification(error.response.data.error));
            yield put(notificationActions.removeOldestNotification());
        }
        yield put(actions.generateClusterInitBundle.failure(error));
    }
}

function* watchLocation() {
    yield takeEveryNewlyMatchedLocation(integrationsPath, getClusterInitBundles);
}

function* watchFetchRequest() {
    while (true) {
        yield take([
            types.FETCH_CLUSTER_INIT_BUNDLES.REQUEST,
            types.GENERATE_CLUSTER_INIT_BUNDLE.SUCCESS,
        ]);
        yield fork(getClusterInitBundles);
    }
}

function* watchGenerateRequest() {
    yield takeLatest(types.GENERATE_CLUSTER_INIT_BUNDLE.REQUEST, generateClusterInitBundle);
}

function* requestFetchRoles() {
    yield put(roleActions.fetchRoles.request());
}

function* watchModalOpen() {
    yield takeLatest(types.START_CLUSTER_INIT_BUNDLE_GENERATION_WIZARD, requestFetchRoles);
}

export default function* integrations() {
    yield all([
        fork(watchLocation),
        fork(watchFetchRequest),
        fork(watchGenerateRequest),
        fork(watchModalOpen),
    ]);
}
