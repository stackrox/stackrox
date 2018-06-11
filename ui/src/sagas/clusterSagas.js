import { takeLatest, call, fork, put, all } from 'redux-saga/effects';

import { integrationsPath, dashboardPath, compliancePath, policiesPath } from 'routePaths';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';
import { fetchClusters } from 'services/ClustersService';
import { actions, types } from 'reducers/clusters';

function* getClusters() {
    try {
        const result = yield call(fetchClusters);
        yield put(actions.fetchClusters.success(result.response));
    } catch (error) {
        yield put(actions.fetchClusters.failure(error));
    }
}

function* watchLocation() {
    const effects = [dashboardPath, integrationsPath, policiesPath, compliancePath].map(path =>
        takeEveryNewlyMatchedLocation(path, getClusters)
    );
    yield all(effects);
}

function* watchFetchRequest() {
    yield takeLatest(types.FETCH_CLUSTERS.REQUEST, getClusters);
}

export default function* clusters() {
    yield all([fork(watchLocation), fork(watchFetchRequest)]);
}
