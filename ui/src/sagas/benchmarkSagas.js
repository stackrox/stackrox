import { all, take, call, fork, put } from 'redux-saga/effects';

import * as service from 'services/BenchmarksService';
import { actions } from 'reducers/benchmarks';
import { types as locationActionTypes } from 'reducers/routes';

const dashboardPath = '/main/dashboard';

export function* getBenchmarks() {
    try {
        const result = yield call(service.fetchUpdatedBenchmarks);
        yield put(actions.fetchBenchmarks.success(result.response));
    } catch (error) {
        yield put(actions.fetchBenchmarks.failure(error));
    }
}

export function* watchDashboardLocation() {
    while (true) {
        const action = yield take(locationActionTypes.LOCATION_CHANGE);
        const { payload: location } = action;
        if (location && location.pathname && location.pathname.startsWith(dashboardPath)) {
            yield fork(getBenchmarks);
        }
    }
}

export default function* benchmarks() {
    yield all([fork(watchDashboardLocation)]);
}
