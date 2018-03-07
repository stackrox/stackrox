import { fork, take } from 'redux-saga/effects';
import { types as locationActionTypes } from 'reducers/routes';
import { getAuthProviders, watchIntegrationsLocation } from './authSagas';

describe('Auth Sagas Test', () => {
    it('Should do a service call to get auth providers when location changes to integrations', () => {
        const gen = watchIntegrationsLocation();
        let { value } = gen.next();
        expect(value).toEqual(take(locationActionTypes.LOCATION_CHANGE));
        ({ value } = gen.next({
            type: locationActionTypes.LOCATION_CHANGE,
            payload: {
                pathname: '/main/integrations'
            }
        }));
        expect(value).toEqual(fork(getAuthProviders));
    });
    it("Shouldn't do a service call to get auth providers when location changes to violations, policies, etc.", () => {
        const gen = watchIntegrationsLocation();
        let { value } = gen.next();
        expect(value).toEqual(take(locationActionTypes.LOCATION_CHANGE));
        ({ value } = gen.next({
            type: locationActionTypes.LOCATION_CHANGE,
            payload: {
                pathname: '/main/violations'
            }
        }));
        expect(value).not.toEqual(fork(getAuthProviders));
    });
});
