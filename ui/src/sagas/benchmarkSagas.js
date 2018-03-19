import { delay } from 'redux-saga';
import { all, take, takeLatest, call, fork, put, select, race } from 'redux-saga/effects';

import * as service from 'services/BenchmarksService';
import { selectors } from 'reducers';
import { actions, types } from 'reducers/benchmarks';
import { types as locationActionTypes } from 'reducers/routes';

const dashboardPath = '/main/dashboard';
const compliancePath = '/main/compliance';

export function* getBenchmarks() {
    try {
        const result = yield call(service.fetchBenchmarks);
        yield put(actions.fetchBenchmarks.success(result.response));
    } catch (error) {
        yield put(actions.fetchBenchmarks.failure(error));
    }
}

export function* getUpdatedBenchmarks() {
    try {
        const result = yield call(service.fetchLastScansByBenchmark);
        yield put(actions.fetchLastScansByBenchmark.success(result.response));
    } catch (error) {
        yield put(actions.fetchLastScansByBenchmark.failure(error));
    }
}

export function* getLastScannedBenchmark(action) {
    try {
        const result = yield call(service.fetchLastScan, action.benchmarkName);
        yield put(actions.fetchLastScan.success(result));
    } catch (error) {
        yield put(actions.fetchLastScan.failure(error));
    }
}

export function* updateBenchmarkSchedule() {
    const schedule = yield select(selectors.getBenchmarkSchedule);
    try {
        if (schedule.hour === '' || schedule.day === '') {
            const newSchedule = Object.assign({}, schedule);
            newSchedule.active = false;
            yield call(service.deleteSchedule, schedule.id);
        } else if (schedule.active) {
            yield call(service.updateSchedule, schedule.benchmarkId, schedule);
        } else {
            const newSchedule = Object.assign({}, schedule);
            newSchedule.active = true;
            yield call(service.createSchedule, newSchedule);
        }
    } catch (error) {
        yield put(actions.fetchLastScan.failure(error));
    }
}

export function* getBenchmarkSchedule({ params: benchmarkId }) {
    try {
        const result = yield call(service.fetchSchedule, benchmarkId);
        yield put(actions.fetchBenchmarkSchedule.success(result));
    } catch (error) {
        yield put(actions.fetchBenchmarkSchedule.failure(error));
    }
}

export function* triggerBenchmarkScan({ params: benchmarkId }) {
    try {
        yield call(service.triggerScan, benchmarkId);
    } catch (error) {
        yield put(actions.triggerScan.failure(error));
    }
}

function* pollBenchmarkScanResults({ params: benchmarkId }) {
    while (true) {
        try {
            console.log(benchmarkId);
            const result = yield call(service.fetchLastScan, benchmarkId);
            yield put(actions.fetchLastScan.success(result));
            yield call(delay, 5000); // poll every 5 sec
        } catch (error) {
            yield put(actions.fetchLastScan.failure(error));
        }
    }
}

function* watchBenchmarkScanResults() {
    while (true) {
        const action = yield take(types.POLL_BENCHMARK_SCAN_RESULTS.START);
        yield race([
            call(pollBenchmarkScanResults, action),
            take(types.POLL_BENCHMARK_SCAN_RESULTS.STOP)
        ]);
    }
}

function* watchUpdateBenchmarkSchedule() {
    yield takeLatest(types.SELECT_BENCHMARK_SCHEDULE_DAY, updateBenchmarkSchedule);
    yield takeLatest(types.SELECT_BENCHMARK_SCHEDULE_HOUR, updateBenchmarkSchedule);
}

function* watchFetchBenchmarkScheduleRequest() {
    yield takeLatest(types.FETCH_BENCHMARK_SCHEDULE.REQUEST, getBenchmarkSchedule);
}

function* watchTriggerBenchmarkScan() {
    yield takeLatest(types.TRIGGER_BENCHMARK_SCAN.REQUEST, triggerBenchmarkScan);
}

export function* watchLocation() {
    while (true) {
        const action = yield take(locationActionTypes.LOCATION_CHANGE);
        const { payload: location } = action;
        if (location && location.pathname && location.pathname.startsWith(dashboardPath)) {
            yield fork(getUpdatedBenchmarks);
        }
        if (location && location.pathname && location.pathname.startsWith(compliancePath)) {
            yield fork(getBenchmarks);
        }
    }
}

export default function* benchmarks() {
    yield all([
        fork(watchLocation),
        fork(watchBenchmarkScanResults),
        fork(watchUpdateBenchmarkSchedule),
        fork(watchFetchBenchmarkScheduleRequest),
        fork(watchTriggerBenchmarkScan)
    ]);
}
