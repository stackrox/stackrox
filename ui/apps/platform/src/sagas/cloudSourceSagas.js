import { all, take, call, fork, put, takeLatest } from 'redux-saga/effects';

import { integrationsPath } from 'routePaths';
import {
    deleteCloudSources as serviceDeleteCloudSources,
    fetchCloudSources as serviceFetchCloudSources,
} from 'services/CloudSourceService';
import { actions, types } from 'reducers/cloudSources';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';

function* getCloudSources() {
    try {
        const result = yield call(serviceFetchCloudSources);
        yield put(actions.fetchCloudSources.success(result.response));
    } catch (error) {
        yield put(actions.fetchCloudSources.failure(error));
    }
}

function* deleteCloudSources({ ids }) {
    try {
        yield call(serviceDeleteCloudSources, ids);
        yield put(actions.fetchCloudSources.request());
    } catch (error) {
        yield put(actions.deleteCloudSources.failure(error));
    }
}

function* watchLocation() {
    yield takeEveryNewlyMatchedLocation(integrationsPath, getCloudSources);
}

function* watchFetchRequest() {
    while (true) {
        yield take([types.FETCH_CLOUD_SOURCES.REQUEST]);
        yield fork(getCloudSources);
    }
}

function* watchDeleteRequest() {
    yield takeLatest(types.DELETE_CLOUD_SOURCES, deleteCloudSources);
}

export default function* integrations() {
    yield all([fork(watchLocation), fork(watchFetchRequest), fork(watchDeleteRequest)]);
}
