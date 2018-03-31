import { delay } from 'redux-saga';
import { take, call, fork, put, cancel } from 'redux-saga/effects';

import { fetchClusters } from 'services/ClustersService';
import { actions } from 'reducers/clusters';
import { types as locationActionTypes } from 'reducers/routes';

const dashboardPath = '/main/dashboard';
const integrationsPath = '/main/integrations';

export function* getClusters() {
    while (true) {
        try {
            const result = yield call(fetchClusters);
            yield put(actions.fetchClusters.success(result.response));
        } catch (error) {
            yield put(actions.fetchClusters.failure(error));
        }
        yield delay(5000); // poll every 5 sec
    }
}

export function* watchLocation() {
    let pollTask;
    while (true) {
        const action = yield take(locationActionTypes.LOCATION_CHANGE);
        const { payload: location } = action;

        if (pollTask) yield cancel(pollTask); // cancel polling in any case
        if (
            location &&
            location.pathname &&
            (location.pathname.startsWith(integrationsPath) ||
                location.pathname.startsWith(dashboardPath))
        ) {
            yield fork(getClusters);
            pollTask = yield fork(getClusters);
        }
    }
}

export default function* benchmarks() {
    yield fork(watchLocation);
}
