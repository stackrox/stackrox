import { takeLatest, call, fork, put, all } from 'redux-saga/effects';

import * as service from 'services/ClustersService';
import { actions, types } from 'reducers/clusters';

function* getClusters() {
    try {
        const result = yield call(service.fetchClusters);
        yield put(actions.fetchClusters.success(result.response));
    } catch (error) {
        yield put(actions.fetchClusters.failure(error));
    }
}

function* watchFetchRequest() {
    yield takeLatest(types.FETCH_CLUSTERS.REQUEST, getClusters);
}

export default function* clusters() {
    yield all([fork(watchFetchRequest)]);
}
