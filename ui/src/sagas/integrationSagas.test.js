import { all, fork, take } from 'redux-saga/effects';
import { types as locationActionTypes } from 'reducers/routes';
import { getNotifiers, getImageIntegrations, watchLocation } from './integrationSagas';

describe('Auth Sagas Test', () => {
    it('Should do a service call to get image integrations, and notifiers when location changes to integrations', () => {
        const gen = watchLocation();
        let { value } = gen.next();
        expect(value).toEqual(take(locationActionTypes.LOCATION_CHANGE));
        ({ value } = gen.next({
            type: locationActionTypes.LOCATION_CHANGE,
            payload: {
                pathname: '/main/integrations'
            }
        }));
        expect(value).toEqual(all([fork(getNotifiers), fork(getImageIntegrations)]));
    });
    it("Shouldn't do a service call to get image integrations, and notifiers when location changes to violations, policies, etc.", () => {
        const gen = watchLocation();
        let { value } = gen.next();
        expect(value).toEqual(take(locationActionTypes.LOCATION_CHANGE));
        ({ value } = gen.next({
            type: locationActionTypes.LOCATION_CHANGE,
            payload: {
                pathname: '/main/violations'
            }
        }));
        expect(value).not.toEqual(all([fork(getNotifiers), fork(getImageIntegrations)]));
    });
});
