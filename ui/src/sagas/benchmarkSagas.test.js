import { fork, take } from 'redux-saga/effects';
import { types as locationActionTypes } from 'reducers/routes';
import { getBenchmarks, watchDashboardLocation } from './benchmarkSagas';

describe('Benchmark Sagas Test', () => {
    it('Should do a service call to get benchmarks when location changes to dashboard', () => {
        const gen = watchDashboardLocation();
        let { value } = gen.next();
        expect(value).toEqual(take(locationActionTypes.LOCATION_CHANGE));
        ({ value } = gen.next({
            type: locationActionTypes.LOCATION_CHANGE,
            payload: {
                pathname: '/main/dashboard'
            }
        }));
        expect(value).toEqual(fork(getBenchmarks));
    });
    it("Shouldn't do a service call to get benchmarks when location changes to violations, policies, etc.", () => {
        const gen = watchDashboardLocation();
        let { value } = gen.next();
        expect(value).toEqual(take(locationActionTypes.LOCATION_CHANGE));
        ({ value } = gen.next({
            type: locationActionTypes.LOCATION_CHANGE,
            payload: {
                pathname: '/main/violations'
            }
        }));
        expect(value).not.toEqual(fork(getBenchmarks));
    });
});
