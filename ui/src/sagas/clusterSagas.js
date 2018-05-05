import { take, takeLatest, call, fork, put, all } from 'redux-saga/effects';

import { fetchClusters } from 'services/ClustersService';
import { actions, types } from 'reducers/clusters';
import { types as locationActionTypes } from 'reducers/routes';

const integrationsPath = '/main/integrations';
const dashboardPath = '/main/dashboard';
const compliancePath = '/main/compliance';

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

        if (
            location &&
            location.pathname &&
            (location.pathname.startsWith(integrationsPath) ||
                location.pathname.startsWith(dashboardPath) ||
                location.pathname.startsWith(compliancePath))
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
