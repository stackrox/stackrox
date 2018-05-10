import { take, takeLatest, call, fork, put, all } from 'redux-saga/effects';

import { fetchClusters } from 'services/ClustersService';
import { actions, types } from 'reducers/clusters';
import { types as locationActionTypes } from 'reducers/routes';

const integrationsPath = '/main/integrations';
const dashboardPath = '/main/dashboard';
const compliancePath = '/main/compliance';
const policiesPath = '/main/policies';

export function* getClusters() {
    try {
        const result = yield call(fetchClusters);
        yield put(actions.fetchClusters.success(result.response));
    } catch (error) {
        yield put(actions.fetchClusters.failure(error));
    }
}

export function* watchLocation() {
    while (true) {
        const action = yield take(locationActionTypes.LOCATION_CHANGE);
        const { payload: location } = action;
        const { pathname } = location;
        if (
            pathname &&
            (pathname.startsWith(integrationsPath) ||
                pathname.startsWith(dashboardPath) ||
                pathname.startsWith(policiesPath) ||
                pathname.startsWith(compliancePath))
        ) {
            yield fork(getClusters);
        }
    }
}

export function* watchFetchRequest() {
    yield takeLatest(types.FETCH_CLUSTERS.REQUEST, getClusters);
}

export default function* clusters() {
    yield all([fork(watchLocation), fork(watchFetchRequest)]);
}
