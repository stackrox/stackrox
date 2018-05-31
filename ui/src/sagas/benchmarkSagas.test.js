import { fork, take, takeLatest, select, call } from 'redux-saga/effects';
import { types as locationActionTypes } from 'reducers/routes';

import {
    getBenchmarks,
    getBenchmarkCheckHostResults,
    getBenchmarksByCluster,
    watchLocation,
    watchBenchmarkCheckHostResults,
    updateBenchmarkSchedule
} from './benchmarkSagas';
import * as service from '../services/BenchmarksService';
import { selectors } from '../reducers';

describe('Benchmark Sagas Test', () => {
    it('Should do a service call to get benchmarks when location changes to dashboard', () => {
        const gen = watchLocation();
        let { value } = gen.next();
        expect(value).toEqual(take(locationActionTypes.LOCATION_CHANGE));
        ({ value } = gen.next({
            type: locationActionTypes.LOCATION_CHANGE,
            payload: {
                pathname: '/main/dashboard/'
            }
        }));
        expect(value).toEqual(fork(getBenchmarksByCluster));
    });
    it('Should do a service call to get benchmarks when location changes to compliance', () => {
        const gen = watchLocation();
        let { value } = gen.next();
        expect(value).toEqual(take(locationActionTypes.LOCATION_CHANGE));
        ({ value } = gen.next({
            type: locationActionTypes.LOCATION_CHANGE,
            payload: {
                pathname: '/main/compliance/'
            }
        }));
        expect(value).toEqual(fork(getBenchmarks));
    });
    it("Shouldn't do a service call to get benchmarks when location changes to violations, policies, etc.", () => {
        const gen = watchLocation();
        let { value } = gen.next();
        expect(value).toEqual(take(locationActionTypes.LOCATION_CHANGE));
        ({ value } = gen.next({
            type: locationActionTypes.LOCATION_CHANGE,
            payload: {
                pathname: '/main/violations/'
            }
        }));
        expect(value).not.toEqual(fork(getBenchmarksByCluster));
    });
    it('Should delete the schedule when no day/hour is selected', () => {
        const removedSchedule = {
            id: '',
            benchmark_id: '',
            benchmark_name: '',
            day: '',
            hour: '',
            active: false,
            timezone_offset: new Date().getTimezoneOffset() / 60
        };
        const gen = updateBenchmarkSchedule();
        let { value } = gen.next();
        expect(value).toEqual(select(selectors.getBenchmarkSchedule));
        ({ value } = gen.next(removedSchedule));
        expect(value).toEqual(call(service.deleteSchedule, removedSchedule.id));
    });
    it('Should update the schedule when a day and time is selected, and the schedule is active', () => {
        const updatedSchedule = {
            id: '',
            benchmarkId: '',
            benchmarkName: '',
            day: 'Friday',
            hour: '5:00 A.M.',
            active: true,
            timezone_offset: new Date().getTimezoneOffset() / 60
        };
        const gen = updateBenchmarkSchedule();
        let { value } = gen.next();
        expect(value).toEqual(select(selectors.getBenchmarkSchedule));
        ({ value } = gen.next(updatedSchedule));
        expect(value).toEqual(
            call(service.updateSchedule, updatedSchedule.benchmarkId, updatedSchedule)
        );
    });
    it('Should create a new schedule when a day and time is selected, and the schedule is not active', () => {
        const newSchedule = {
            id: '',
            benchmarkId: '',
            benchmarkName: '',
            day: 'Friday',
            hour: '5:00 A.M.',
            active: false,
            timezone_offset: new Date().getTimezoneOffset() / 60
        };
        const gen = updateBenchmarkSchedule();
        let { value } = gen.next();
        expect(value).toEqual(select(selectors.getBenchmarkSchedule));
        ({ value } = gen.next(newSchedule));
        const modifiedSchedule = Object.assign(newSchedule, { active: true });
        expect(value).toEqual(call(service.createSchedule, modifiedSchedule));
    });

    it('should fetch benchmark details on a Compliance page with benchmark scan selected', () => {
        let gen = watchBenchmarkCheckHostResults();
        let { value } = gen.next();
        expect(value).toEqual(
            takeLatest(
                'benchmarks/FETCH_BENCHMARK_CHECK_HOST_RESULTS_REQUEST',
                getBenchmarkCheckHostResults
            )
        );

        const benchmark = {
            scanId: '123',
            checkName: 'CIS 1.1'
        };

        gen = getBenchmarkCheckHostResults({ params: benchmark });
        ({ value } = gen.next());
        expect(value).toEqual(call(service.fetchBenchmarkCheckHostResults, benchmark));
    });
});
