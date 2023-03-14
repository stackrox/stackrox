import { all, take, call, fork, put } from 'redux-saga/effects';

import { integrationsPath } from 'routePaths';
import * as service from 'services/ClustersService';
import { actions, types } from 'reducers/clusterInitBundles';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';

function* getClusterInitBundles() {
    try {
        const result = yield call(service.fetchClusterInitBundles);
        yield put(actions.fetchClusterInitBundles.success(result.response));
    } catch (error) {
        yield put(actions.fetchClusterInitBundles.failure(error));
    }
}

function* watchLocation() {
    yield takeEveryNewlyMatchedLocation(integrationsPath, getClusterInitBundles);
}

function* watchFetchRequest() {
    while (true) {
        yield take([types.FETCH_CLUSTER_INIT_BUNDLES.REQUEST]);
        yield fork(getClusterInitBundles);
    }
}

export default function* integrations() {
    yield all([fork(watchLocation), fork(watchFetchRequest)]);
}
