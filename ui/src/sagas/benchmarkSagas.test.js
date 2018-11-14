import { select, call } from 'redux-saga/effects';
import { expectSaga } from 'redux-saga-test-plan';
import { dynamic } from 'redux-saga-test-plan/providers';

import { actions as benchmarkActions, types as benchmarkTypes } from 'reducers/benchmarks';
import { types as dashboardTypes } from 'reducers/dashboard';
import saga from './benchmarkSagas';
import { selectors } from '../reducers';
import * as service from '../services/BenchmarksService';
import createLocationChange from './sagaTestUtils';

const dashboardSearchOptions = [
    {
        value: 'Cluster:',
        label: 'Cluster:',
        type: 'categoryOption'
    },
    {
        value: 'cluster1',
        label: 'cluster1'
    }
];

const dashboardSearchQuery = {
    query: 'Cluster:cluster1'
};

const clusters = [
    {
        benchmarks: [],
        clusterId: 'b5c6f9a2-c80e-4dea-aca2-501420931c67',
        clusterName: 'cluster1'
    },
    {
        benchmarks: [],
        clusterId: 'f5c5f9a2-c80e-4dea-aca2-531450931c67',
        clusterName: 'cluster2'
    }
];
const filteredClusters = [
    {
        benchmarks: [],
        clusterId: 'b5c6f9a2-c80e-4dea-aca2-501420931c67',
        clusterName: 'cluster1'
    }
];

describe('Benchmark Sagas Test', () => {
    it('should get an unfiltered list of clusters on Dashboard page when searching', () => {
        const fetchMock = jest.fn().mockReturnValue(clusters);

        return expectSaga(saga)
            .provide([
                [select(selectors.getDashboardSearchOptions), []],
                [call(service.fetchBenchmarksByCluster, { query: '' }), dynamic(fetchMock)]
            ])
            .dispatch(createLocationChange('/main/dashboard'))
            .dispatch({
                type: dashboardTypes.SET_SEARCH_OPTIONS,
                payload: { options: [] }
            })
            .put(benchmarkActions.fetchBenchmarksByCluster.success(clusters))
            .silentRun();
    });

    it('should get a filtered list of clusters on Dashboard page when searching', () => {
        const fetchMock = jest.fn().mockReturnValueOnce(filteredClusters);

        return expectSaga(saga)
            .provide([
                [select(selectors.getDashboardSearchOptions), dashboardSearchOptions],
                [call(service.fetchBenchmarksByCluster, dashboardSearchQuery), dynamic(fetchMock)]
            ])
            .dispatch(createLocationChange('/main/dashboard'))
            .dispatch({
                type: dashboardTypes.SET_SEARCH_OPTIONS,
                payload: { options: dashboardSearchOptions }
            })
            .put(benchmarkActions.fetchBenchmarksByCluster.success(filteredClusters))
            .silentRun();
    });

    it('should do a service call to get benchmarks when location changes to Compliance page', () => {
        const benchmarks = [
            {
                id: '27ec04c3-0d28-4f5f-82bf-5f746250f11b',
                name: 'CIS Swarm v1.1.0 Benchmark',
                editable: false,
                checks: []
            }
        ];
        const fetchMock = jest.fn().mockReturnValueOnce(benchmarks);

        return expectSaga(saga)
            .provide([[call(service.fetchBenchmarks), dynamic(fetchMock)]])
            .dispatch(createLocationChange('/main/compliance'))
            .put(benchmarkActions.fetchBenchmarks.success(benchmarks))
            .silentRun();
    });

    it('should not fetch benchmarks when location changes to violations, policies, etc.', () => {
        const fetchBenchmarksByClusterMock = jest.fn();
        const fetchBenchmarksMock = jest.fn();

        return expectSaga(saga)
            .provide([[call(service.fetchBenchmarks), dynamic(fetchBenchmarksMock)]])
            .dispatch(createLocationChange('/main/violations'))
            .dispatch(createLocationChange('/main/policies'))
            .dispatch(createLocationChange('/main/integrations'))
            .silentRun()
            .then(() => {
                expect(fetchBenchmarksByClusterMock.mock.calls.length).toBe(0);
                expect(fetchBenchmarksMock.mock.calls.length).toBe(0);
            });
    });

    it('should delete the schedule when no day/hour is selected', () => {
        const schedule = {
            id: undefined,
            benchmarkId: '27ec04c3-0d28-4f5f-82bf-5f746250f11b',
            benchmarkName: 'CIS Swarm v1.1.0 Benchmark',
            day: '',
            hour: '',
            timezone_offset: new Date().getTimezoneOffset() / 60
        };

        return expectSaga(saga)
            .provide([[select(selectors.getBenchmarkSchedule), schedule]])
            .dispatch(createLocationChange('/main/compliance'))
            .dispatch({
                type: benchmarkTypes.SELECT_BENCHMARK_SCHEDULE_DAY,
                benchmarkId: '27ec04c3-0d28-4f5f-82bf-5f746250f11b',
                benchmarkName: 'CIS Swarm v1.1.0 Benchmark',
                value: 'None'
            })
            .call.like({ fn: service.deleteSchedule })
            .silentRun();
    });

    it('should update the schedule when a day and time is selected, and the schedule is active', () => {
        const schedule = {
            id: 'scheduleId',
            benchmarkId: '27ec04c3-0d28-4f5f-82bf-5f746250f11b',
            benchmarkName: 'CIS Swarm v1.1.0 Benchmark',
            clusterId: 'b5c6f9a2-c80e-4dea-aca2-501420931c67',
            day: 'Monday',
            hour: '02:00 AM',
            timezone_offset: 7
        };

        return expectSaga(saga)
            .provide([[select(selectors.getBenchmarkSchedule), schedule]])
            .dispatch(createLocationChange('/main/compliance'))
            .dispatch({
                type: benchmarkTypes.SELECT_BENCHMARK_SCHEDULE_DAY,
                benchmarkId: '27ec04c3-0d28-4f5f-82bf-5f746250f11b',
                benchmarkName: 'CIS Swarm v1.1.0 Benchmark',
                value: 'Monday'
            })
            .dispatch({
                type: benchmarkTypes.SELECT_BENCHMARK_SCHEDULE_DAY,
                benchmarkId: '27ec04c3-0d28-4f5f-82bf-5f746250f11b',
                benchmarkName: 'CIS Swarm v1.1.0 Benchmark',
                value: '02:00 AM',
                clusterId: 'b5c6f9a2-c80e-4dea-aca2-501420931c67'
            })
            .call.like({ fn: service.updateSchedule })
            .silentRun();
    });

    it('Should create a new schedule when a day and time is selected, and the schedule is not active', () => {
        const schedule = {
            id: undefined,
            active: false,
            benchmarkId: '27ec04c3-0d28-4f5f-82bf-5f746250f11b',
            benchmarkName: 'CIS Swarm v1.1.0 Benchmark',
            clusterId: 'b5c6f9a2-c80e-4dea-aca2-501420931c67',
            day: 'Monday',
            hour: '02:00 AM',
            timezone_offset: 7
        };

        return expectSaga(saga)
            .provide([[select(selectors.getBenchmarkSchedule), schedule]])
            .dispatch(createLocationChange('/main/compliance'))
            .dispatch({
                type: benchmarkTypes.SELECT_BENCHMARK_SCHEDULE_DAY,
                benchmarkId: '27ec04c3-0d28-4f5f-82bf-5f746250f11b',
                benchmarkName: 'CIS Swarm v1.1.0 Benchmark',
                value: 'Monday'
            })
            .dispatch({
                type: benchmarkTypes.SELECT_BENCHMARK_SCHEDULE_DAY,
                benchmarkId: '27ec04c3-0d28-4f5f-82bf-5f746250f11b',
                benchmarkName: 'CIS Swarm v1.1.0 Benchmark',
                value: '02:00 AM',
                clusterId: 'b5c6f9a2-c80e-4dea-aca2-501420931c67'
            })
            .call.like({ fn: service.createSchedule })
            .silentRun();
    });
});
