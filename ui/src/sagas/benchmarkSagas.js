import { delay } from 'redux-saga';
import { all, take, takeLatest, call, fork, put, select, race } from 'redux-saga/effects';

import * as service from 'services/BenchmarksService';
import { selectors } from 'reducers';
import { actions as benchmarkActions, types } from 'reducers/benchmarks';
import { types as locationActionTypes } from 'reducers/routes';

const dashboardPath = '/main/dashboard';
const compliancePath = '/main/compliance';

export function* getBenchmarks() {
    try {
        const result = yield call(service.fetchBenchmarks);
        yield put(benchmarkActions.fetchBenchmarks.success(result));
    } catch (error) {
        yield put(benchmarkActions.fetchBenchmarks.failure(error));
    }
}

export function* getBenchmarksByCluster() {
    try {
        const result = yield call(service.fetchBenchmarksByCluster);
        yield put(benchmarkActions.fetchBenchmarksByCluster.success(result));
    } catch (error) {
        yield put(benchmarkActions.fetchBenchmarksByCluster.failure(error));
    }
}

export function* updateBenchmarkSchedule() {
    const schedule = Object.assign({}, yield select(selectors.getBenchmarkSchedule));
    try {
        if (schedule.hour === '' || schedule.day === '') {
            if (schedule.hour === schedule.day) {
                schedule.active = false;
                yield call(service.deleteSchedule, schedule.id);
            }
        } else if (schedule.active) {
            yield call(service.updateSchedule, schedule.id, schedule);
        } else {
            schedule.active = true;
            yield call(service.createSchedule, schedule);
        }
    } catch (error) {
        yield put(benchmarkActions.fetchLastScan.failure(error));
    }
}

export function* getBenchmarkSchedule({ params: benchmark }) {
    try {
        const result = yield call(service.fetchSchedule, benchmark);
        yield put(benchmarkActions.fetchBenchmarkSchedule.success(result));
    } catch (error) {
        yield put(benchmarkActions.fetchBenchmarkSchedule.failure(error));
    }
}

export function* triggerBenchmarkScan({ params: benchmark }) {
    try {
        yield call(service.triggerScan, benchmark);
    } catch (error) {
        yield put(benchmarkActions.triggerScan.failure(error));
    }
}

function* pollBenchmarkScanResults({ params: benchmark }) {
    while (true) {
        try {
            const result = yield call(service.fetchLastScan, benchmark);
            yield put(benchmarkActions.fetchLastScan.success(result));
        } catch (error) {
            yield put(benchmarkActions.fetchLastScan.failure(error));
        }
        yield call(delay, 5000); // poll every 5 sec
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
            yield fork(getBenchmarksByCluster);
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
