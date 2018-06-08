import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';
import { createPollingActionTypes, createPollingActions } from 'utils/pollingReduxRoutines';

const initialBenchmarkSchedule = {
    id: '',
    benchmarkId: '',
    benchmarkName: '',
    day: '',
    hour: '',
    timezone_offset: new Date().getTimezoneOffset() / 60
};

// Action types

export const types = {
    POLL_BENCHMARK_SCAN_RESULTS: createPollingActionTypes('benchmarks/POLL_BENCHMARK_SCAN_RESULTS'),
    SELECT_BENCHMARK_SCHEDULE_DAY: 'benchmarks/SELECT_BENCHMARK_SCHEDULE_DAY',
    SELECT_BENCHMARK_SCHEDULE_HOUR: 'benchmarks/SELECT_BENCHMARK_SCHEDULE_HOUR',
    SELECT_BENCHMARK_SCAN_RESULT: 'benchmarks/SELECT_BENCHMARK_SCAN_RESULT',
    SELECT_BENCHMARK_HOST_RESULT: 'benchmarks/SELECT_BENCHMARK_HOST_RESULT',
    TRIGGER_BENCHMARK_SCAN: createFetchingActionTypes('benchmarks/TRIGGER_BENCHMARK_SCAN'),
    FETCH_BENCHMARK_SCHEDULE: createFetchingActionTypes('benchmarks/FETCH_BENCHMARK_SCHEDULE'),
    FETCH_BENCHMARKS: createFetchingActionTypes('benchmarks/FETCH_BENCHMARKS'),
    FETCH_BENCHMARK_CHECK_HOST_RESULTS: createFetchingActionTypes(
        'benchmarks/FETCH_BENCHMARK_CHECK_HOST_RESULTS'
    ),
    FETCH_BENCHMARKS_BY_CLUSTER: createFetchingActionTypes(
        'benchmarks/FETCH_BENCHMARKS_BY_CLUSTER'
    ),
    FETCH_LAST_SCANS_BY_BENCHMARK: createFetchingActionTypes(
        'benchmarks/FETCH_LAST_SCANS_BY_BENCHMARK'
    ),
    FETCH_LAST_SCAN: createFetchingActionTypes('benchmarks/FETCH_LAST_SCAN')
};

// Actions

export const actions = {
    pollBenchmarkScanResults: createPollingActions(types.POLL_BENCHMARK_SCAN_RESULTS),
    selectBenchmarkScheduleDay: (benchmarkId, benchmarkName, value, clusterId) => ({
        type: types.SELECT_BENCHMARK_SCHEDULE_DAY,
        benchmarkId,
        benchmarkName,
        value,
        clusterId
    }),
    selectBenchmarkScheduleHour: (benchmarkId, benchmarkName, value, clusterId) => ({
        type: types.SELECT_BENCHMARK_SCHEDULE_HOUR,
        benchmarkId,
        benchmarkName,
        value,
        clusterId
    }),
    selectBenchmarkScanResult: benchmarkScanResult => ({
        type: types.SELECT_BENCHMARK_SCAN_RESULT,
        benchmarkScanResult
    }),
    selectBenchmarkHostResult: benchmarkHostResult => ({
        type: types.SELECT_BENCHMARK_HOST_RESULT,
        benchmarkHostResult
    }),
    triggerBenchmarkScan: createFetchingActions(types.TRIGGER_BENCHMARK_SCAN),
    fetchBenchmarkSchedule: createFetchingActions(types.FETCH_BENCHMARK_SCHEDULE),
    fetchBenchmarks: createFetchingActions(types.FETCH_BENCHMARKS),
    fetchBenchmarkCheckHostResults: createFetchingActions(types.FETCH_BENCHMARK_CHECK_HOST_RESULTS),
    fetchBenchmarksByCluster: createFetchingActions(types.FETCH_BENCHMARKS_BY_CLUSTER),
    fetchLastScan: createFetchingActions(types.FETCH_LAST_SCAN)
};

// Reducers

const benchmarks = (state = [], action) => {
    if (action.type === types.FETCH_BENCHMARKS.SUCCESS) {
        const { response } = action;
        return isEqual(response, state) ? state : response;
    }
    return state;
};

const benchmarkCheckHostResults = (state = null, action) => {
    if (action.type === types.FETCH_BENCHMARK_CHECK_HOST_RESULTS.SUCCESS) {
        const { response } = action;
        return isEqual(response, state) ? state : response;
    }
    return state;
};

const benchmarksByCluster = (state = [], action) => {
    if (action.type === types.FETCH_BENCHMARKS_BY_CLUSTER.SUCCESS) {
        const { response } = action;
        return isEqual(response, state) ? state : response;
    }
    return state;
};

const lastScan = (state = {}, action) => {
    if (action.type === types.FETCH_LAST_SCAN.SUCCESS) {
        const { response } = action;
        return isEqual(response, state) ? state : response;
    }
    return state;
};

const selectedBenchmarkScanResult = (state = null, action) => {
    if (action.type === types.SELECT_BENCHMARK_SCAN_RESULT) {
        return action.benchmarkScanResult;
    }
    return state;
};

const selectedBenchmarkHostResult = (state = null, action) => {
    if (action.type === types.SELECT_BENCHMARK_HOST_RESULT) {
        return action.benchmarkHostResult;
    }
    return state;
};

const benchmarkSchedule = (state = initialBenchmarkSchedule, action) => {
    if (action.type === types.FETCH_BENCHMARK_SCHEDULE.SUCCESS) {
        const schedule = Object.assign({}, action.response.schedules[0]);
        schedule.timezone_offset = new Date().getTimezoneOffset() / 60;
        return isEqual(schedule, state) ? state : schedule;
    }
    const addToClusters = (scheduleClusterIds, clusterId) => {
        if (!scheduleClusterIds) return [clusterId];
        if (!scheduleClusterIds.includes(clusterId)) return [...scheduleClusterIds, clusterId];
        return scheduleClusterIds;
    };
    if (action.type === types.SELECT_BENCHMARK_SCHEDULE_DAY) {
        const schedule = Object.assign({}, state);
        schedule.benchmarkId = action.benchmarkId;
        schedule.benchmarkName = action.benchmarkName;
        schedule.clusterIds = addToClusters(schedule.clusterIds, action.clusterId);
        if (action.value === 'None') {
            schedule.day = '';
            schedule.hour = '';
        } else {
            schedule.day = action.value;
        }
        return schedule;
    }
    if (action.type === types.SELECT_BENCHMARK_SCHEDULE_HOUR) {
        const schedule = Object.assign({}, state);
        schedule.benchmarkId = action.benchmarkId;
        schedule.benchmarkName = action.benchmarkName;
        schedule.clusterIds = addToClusters(schedule.clusterIds, action.clusterId);
        schedule.hour = action.value;
        return schedule;
    }
    return state;
};

const reducer = combineReducers({
    benchmarks,
    benchmarkCheckHostResults,
    benchmarksByCluster,
    lastScan,
    benchmarkSchedule,
    selectedBenchmarkScanResult,
    selectedBenchmarkHostResult
});

export default reducer;

// Selectors

const getBenchmarks = state => state.benchmarks;
const getBenchmarkCheckHostResults = state => state.benchmarkCheckHostResults;
const getBenchmarksByCluster = state => state.benchmarksByCluster;
const getLastScan = state => state.lastScan;
const getBenchmarkSchedule = state => state.benchmarkSchedule;
const getSelectedBenchmarkScanResult = state => state.selectedBenchmarkScanResult;
const getSelectedBenchmarkHostResult = state => state.selectedBenchmarkHostResult;

export const selectors = {
    getBenchmarks,
    getBenchmarkCheckHostResults,
    getBenchmarksByCluster,
    getLastScan,
    getBenchmarkSchedule,
    getSelectedBenchmarkScanResult,
    getSelectedBenchmarkHostResult
};
