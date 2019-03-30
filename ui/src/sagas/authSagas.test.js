import { select, call } from 'redux-saga/effects';
import { push } from 'react-router-redux';
import { expectSaga } from 'redux-saga-test-plan';
import { dynamic, throwError } from 'redux-saga-test-plan/providers';

import { selectors } from 'reducers';
import { actions, AUTH_STATUS } from 'reducers/auth';
import { LICENSE_STATUS } from 'reducers/license';
import * as AuthService from 'services/AuthService';
import * as LicenseService from 'services/LicenseService';
import saga from './authSagas';
import createLocationChange from './sagaTestUtils';

const createStateSelectors = (authProviders = [], authStatus = AUTH_STATUS.LOADING) => [
    [select(selectors.getAuthProviders), authProviders],
    [select(selectors.getAuthStatus), authStatus]
];

describe('Auth Sagas', () => {
    it('should get and put auth providers when on integrations page', () => {
        const authProviders = [{ name: 'ap1', validated: true }, { name: 'ap1', validated: false }];
        const fetchMock = jest
            .fn()
            .mockReturnValueOnce({ response: [] })
            .mockReturnValueOnce({ response: authProviders });

        return expectSaga(saga)
            .provide([
                ...createStateSelectors(),
                [call(AuthService.fetchAuthProviders), dynamic(fetchMock)]
            ])
            .put(actions.fetchAuthProviders.success(authProviders))
            .dispatch(createLocationChange('/')) // first location change will also trigger auth providers fetching
            .dispatch(createLocationChange('/main/integrations'))
            .silentRun();
    });

    it('should not do a service call to get auth providers when location changes to violations, policies, etc.', () => {
        const fetchMock = jest.fn().mockReturnValue({ response: [] });
        return expectSaga(saga)
            .provide([
                ...createStateSelectors(),
                [call(AuthService.fetchAuthProviders), dynamic(fetchMock)]
            ])
            .dispatch(createLocationChange('/'))
            .dispatch(createLocationChange('/main/policies'))
            .dispatch(createLocationChange('/main/violations'))
            .silentRun()
            .then(() => {
                expect(fetchMock.mock.calls.length).toBe(1); // always called at the beginning
            });
    });

    it('should log out the anonymous user if auth provider was added', () =>
        expectSaga(saga)
            .provide([
                ...createStateSelectors(
                    [{ name: 'ap1', validated: true }],
                    AUTH_STATUS.ANONYMOUS_ACCESS
                ),
                [
                    call(AuthService.fetchAuthProviders),
                    { response: [{ name: 'ap1', validated: true }] }
                ],
                [call(AuthService.isTokenPresent), false]
            ])
            .put(actions.logout())
            .dispatch(createLocationChange('/'))
            .silentRun());

    it('should not log out the anonymous user if unvalidated auth provider was added', () => {
        const logoutMock = jest.fn();

        return expectSaga(saga)
            .provide([
                ...createStateSelectors(
                    [{ name: 'ap1', validated: false }],
                    AUTH_STATUS.ANONYMOUS_ACCESS
                ),
                [
                    call(AuthService.fetchAuthProviders),
                    { response: [{ name: 'ap1', validated: false }] }
                ],
                [call(AuthService.isTokenPresent), false],
                [call(actions.logout), dynamic(logoutMock)]
            ])
            .dispatch(createLocationChange('/'))
            .silentRun()
            .then(() => {
                expect(logoutMock.mock.calls.length).toBe(0);
            });
    });

    it('should check auth status with existing valid token and login the user', () =>
        expectSaga(saga)
            .provide([
                ...createStateSelectors([{ name: 'ap1', validated: true }]),
                [
                    call(AuthService.fetchAuthProviders),
                    { response: [{ name: 'ap1', validated: true }] }
                ],
                [call(AuthService.isTokenPresent), true],
                [call(AuthService.fetchAuthStatus), 'ok']
            ])
            .put(actions.login())
            .dispatch(createLocationChange('/'))
            .silentRun());

    it('should check auth status with existing invalid token and logout the user', () =>
        expectSaga(saga)
            .provide([
                ...createStateSelectors([{ name: 'ap1', validated: true }]),
                [
                    call(AuthService.fetchAuthProviders),
                    { response: [{ name: 'ap1', validated: true }] }
                ],
                [call(AuthService.isTokenPresent), true],
                [call(AuthService.fetchAuthStatus), throwError(new Error('401'))]
            ])
            .put(actions.logout())
            .dispatch(createLocationChange('/'))
            .silentRun());

    it('should grant anonymous access w/o auth providers and clear any existing token', () => {
        const clearTokenMock = jest.fn();
        return expectSaga(saga)
            .provide([
                ...createStateSelectors(),
                [call(AuthService.fetchAuthProviders), { response: [] }],
                [call(AuthService.isTokenPresent), true],
                [call(AuthService.clearAccessToken), dynamic(clearTokenMock)]
            ])
            .put(actions.grantAnonymousAccess())
            .dispatch(createLocationChange('/'))
            .silentRun()
            .then(() => {
                expect(clearTokenMock.mock.calls.length).toBe(1);
            });
    });

    it('should grant anonymous access w/o auth providers and clear the token', () => {
        const clearTokenMock = jest.fn();
        return expectSaga(saga)
            .provide([
                ...createStateSelectors(),
                [call(AuthService.fetchAuthProviders), { response: [] }],
                [call(AuthService.isTokenPresent), true],
                [call(AuthService.clearAccessToken), dynamic(clearTokenMock)]
            ])
            .put(actions.grantAnonymousAccess())
            .dispatch(createLocationChange('/'))
            .silentRun()
            .then(() => {
                expect(clearTokenMock.mock.calls.length).toBe(1);
            });
    });

    it('should clear the token when user logs out', () => {
        const clearTokenMock = jest.fn();
        return expectSaga(saga)
            .provide([
                ...createStateSelectors([{ name: 'ap1', validated: true }], AUTH_STATUS.LOGGED_IN),
                [
                    call(AuthService.fetchAuthProviders),
                    { response: [{ name: 'ap1', validated: true }] }
                ],
                [call(AuthService.isTokenPresent), true],
                [call(AuthService.clearAccessToken), dynamic(clearTokenMock)]
            ])
            .dispatch(createLocationChange('/'))
            .dispatch(actions.logout())
            .silentRun()
            .then(() => {
                expect(clearTokenMock.mock.calls.length).toBe(1);
            });
    });

    it('should store the previous location after being redirected to login page', () => {
        const storeLocationMock = jest.fn();
        const from = '/from';
        return expectSaga(saga)
            .provide([
                ...createStateSelectors(),
                [call(AuthService.fetchAuthProviders), { response: [] }],
                [call(AuthService.storeRequestedLocation, from), dynamic(storeLocationMock)]
            ])
            .dispatch(createLocationChange('/'))
            .dispatch(createLocationChange('/login', from))
            .silentRun()
            .then(() => {
                expect(storeLocationMock.mock.calls.length).toBe(1);
            });
    });

    it('should handle OIDC redirect and restore previous location', () => {
        const storeAccessTokenMock = jest.fn();
        const token = 'my-token';
        const serverState = 'provider-prefix:client-state';
        const exchangedToken = 'my-rox-token';
        const requestedLocation = '/my-location';
        return expectSaga(saga)
            .provide([
                ...createStateSelectors(),
                [call(AuthService.fetchAuthProviders), { response: [] }],
                [
                    call(AuthService.exchangeAuthToken, token, 'oidc', serverState),
                    { token: exchangedToken }
                ],
                [call(AuthService.storeAccessToken, exchangedToken), dynamic(storeAccessTokenMock)],
                [call(AuthService.getAndClearRequestedLocation), requestedLocation],
                [
                    call(LicenseService.fetchLicenses),
                    {
                        response: { licenses: [{ status: LICENSE_STATUS.VALID }] }
                    }
                ]
            ])
            .put(push(requestedLocation))
            .dispatch(
                createLocationChange(
                    '/auth/response/oidc',
                    null,
                    `#id_token=${token}&state=${serverState}`
                )
            )
            .silentRun()
            .then(() => {
                expect(storeAccessTokenMock.mock.calls.length).toBe(1);
            });
    });

    it('should logout in case of 401 HTTP error', () =>
        expectSaga(saga)
            .provide([
                ...createStateSelectors([{ name: 'ap1', validated: true }], AUTH_STATUS.LOGGED_IN),
                [
                    call(AuthService.fetchAuthProviders),
                    { response: [{ name: 'ap1', validated: true }] }
                ],
                [call(AuthService.isTokenPresent), true]
            ])
            .put(actions.logout())
            .dispatch(createLocationChange('/'))
            .dispatch(actions.handleAuthHttpError(new AuthService.AuthHttpError('error', 401)))
            .silentRun());

    it('should ignore 403 HTTP error', () =>
        expectSaga(saga)
            .provide([
                ...createStateSelectors([{ name: 'ap1', validated: true }], AUTH_STATUS.LOGGED_IN),
                [
                    call(AuthService.fetchAuthProviders),
                    { response: [{ name: 'ap1', validated: true }] }
                ],
                [call(AuthService.isTokenPresent), true]
            ])
            .not.put(actions.logout())
            .dispatch(createLocationChange('/'))
            .dispatch(actions.handleAuthHttpError(new AuthService.AuthHttpError('error', 403)))
            .silentRun());
});
