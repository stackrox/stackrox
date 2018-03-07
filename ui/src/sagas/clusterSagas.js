import { take, call, fork, put } from 'redux-saga/effects';

import fetchClusters from 'services/ClustersService';
import { actions } from 'reducers/clusters';
import { types as locationActionTypes } from 'reducers/routes';

const dashboardPath = '/main/dashboard';
const integrationsPath = '/main/integrations';

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
                location.pathname.startsWith(dashboardPath))
        ) {
            yield fork(getClusters);
        }
    }
}

export default function* benchmarks() {
    yield fork(watchLocation);
}
