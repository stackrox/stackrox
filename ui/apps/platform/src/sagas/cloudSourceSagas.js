import { all, call, fork, put, take } from 'redux-saga/effects';

import { integrationsPath } from 'routePaths';
import { fetchCloudSources as serviceFetchCloudSources } from 'services/CloudSourceService';
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

function* watchLocation() {
    yield takeEveryNewlyMatchedLocation(integrationsPath, getCloudSources);
}

function* watchFetchRequest() {
    while (true) {
        yield take([types.FETCH_CLOUD_SOURCES.REQUEST]);
        yield fork(getCloudSources);
    }
}

export default function* integrations() {
    yield all([fork(watchLocation), fork(watchFetchRequest)]);
}
