import { fork, take } from 'redux-saga/effects';
import { types as locationActionTypes } from 'reducers/routes';
import { getClusters, watchDashboardLocation } from './clusterSagas';

describe('Cluster Sagas Test', () => {
    it('Should do a service call to get clusters when location changes to dashboard', () => {
        const gen = watchDashboardLocation();
        let { value } = gen.next();
        expect(value).toEqual(take(locationActionTypes.LOCATION_CHANGE));
        ({ value } = gen.next({
            type: locationActionTypes.LOCATION_CHANGE,
            payload: {
                pathname: '/main/dashboard'
            }
        }));
        expect(value).toEqual(fork(getClusters));
    });
    it("Shouldn't do a service call to get clusters when location changes to violations, policies, etc.", () => {
        const gen = watchDashboardLocation();
        let { value } = gen.next();
        expect(value).toEqual(take(locationActionTypes.LOCATION_CHANGE));
        ({ value } = gen.next({
            type: locationActionTypes.LOCATION_CHANGE,
            payload: {
                pathname: '/main/violations'
            }
        }));
        expect(value).not.toEqual(fork(getClusters));
    });
});
