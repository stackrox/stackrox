import { select, call } from 'redux-saga/effects';
import { push } from 'connected-react-router';
import { expectSaga } from 'redux-saga-test-plan';
import { dynamic, throwError } from 'redux-saga-test-plan/providers';

import { selectors } from 'reducers';
import { actions, AUTH_STATUS } from 'reducers/auth';
import * as AuthService from 'services/AuthService';
import { fetchUserRolePermissions } from 'services/RolesService';
import saga from './authSagas';
import createLocationChange from './sagaTestUtils';

const createStateSelectors = (authProviders = [], authStatus = AUTH_STATUS.LOADING) => [
    [select(selectors.getLoginAuthProviders), authProviders],
    [select(selectors.getAuthProviders), authProviders],
    [select(selectors.getAuthStatus), authStatus],
];

describe('Auth Sagas', () => {
    it('should not do a service call to get auth providers when location changes to violations, policies, etc.', () => {
        const fetchMock = jest.fn().mockReturnValue({ response: [] });
        return expectSaga(saga)
            .provide([
                ...createStateSelectors(),
                [call(AuthService.fetchLoginAuthProviders), dynamic(fetchMock)],
                [call(AuthService.logout), null],
                [call(fetchUserRolePermissions), { response: {} }],
                [call(AuthService.fetchAvailableProviderTypes), { response: [] }],
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
                ...createStateSelectors([{ name: 'ap1' }], AUTH_STATUS.ANONYMOUS_ACCESS),
                [call(AuthService.fetchLoginAuthProviders), { response: [{ name: 'ap1' }] }],
                [call(AuthService.getAccessToken), null],
                [call(AuthService.logout), null],
                [call(fetchUserRolePermissions), { response: {} }],
                [call(AuthService.fetchAvailableProviderTypes), { response: [] }],
            ])
            .put(actions.logout())
            .dispatch(createLocationChange('/'))
            .silentRun());

    it('should check auth status with existing valid token and login the user', () =>
        expectSaga(saga)
            .provide([
                ...createStateSelectors([{ name: 'ap1' }]),
                [call(AuthService.fetchLoginAuthProviders), { response: [{ name: 'ap1' }] }],
                [call(AuthService.getAccessToken), 'my-token'],
                [call(AuthService.getAuthStatus), 'ok'],
                [call(fetchUserRolePermissions), { response: {} }],
                [call(AuthService.fetchAvailableProviderTypes), { response: [] }],
            ])
            .put(actions.login('ok'))
            .dispatch(createLocationChange('/'))
            .silentRun());

    it('should check auth status with existing invalid token and logout the user', () =>
        expectSaga(saga)
            .provide([
                ...createStateSelectors([{ name: 'ap1' }]),
                [call(AuthService.fetchLoginAuthProviders), { response: [{ name: 'ap1' }] }],
                [call(AuthService.getAccessToken), 'my-token'],
                [call(AuthService.getAuthStatus), throwError(new Error('401'))],
                [call(AuthService.logout), null],
                [call(fetchUserRolePermissions), { response: {} }],
                [call(AuthService.fetchAvailableProviderTypes), { response: [] }],
            ])
            .put(actions.logout())
            .dispatch(createLocationChange('/'))
            .silentRun());

    it('should clear the token when user logs out', () => {
        const logout = jest.fn();
        return expectSaga(saga)
            .provide([
                ...createStateSelectors([{ name: 'ap1' }], AUTH_STATUS.LOGGED_IN),
                [call(AuthService.fetchLoginAuthProviders), { response: [{ name: 'ap1' }] }],
                [call(AuthService.getAccessToken), 'my-token'],
                [call(AuthService.logout), dynamic(logout)],
                [call(fetchUserRolePermissions), { response: {} }],
                [call(AuthService.fetchAvailableProviderTypes), { response: [] }],
            ])
            .dispatch(createLocationChange('/'))
            .dispatch(actions.logout())
            .silentRun()
            .then(() => {
                expect(logout.mock.calls.length).toBe(1);
            });
    });

    it('should store the previous location after being redirected to login page', () => {
        const storeLocationMock = jest.fn();
        const from = '/from';
        return expectSaga(saga)
            .provide([
                ...createStateSelectors(),
                [call(AuthService.fetchLoginAuthProviders), { response: [] }],
                [call(AuthService.logout), null],
                [call(AuthService.storeRequestedLocation, from), dynamic(storeLocationMock)],
                [call(fetchUserRolePermissions), { response: {} }],
                [call(AuthService.fetchAvailableProviderTypes), { response: [] }],
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
                [call(AuthService.fetchLoginAuthProviders), { response: [] }],
                [
                    call(AuthService.exchangeAuthToken, `#id_token=${token}`, 'oidc', serverState),
                    { token: exchangedToken },
                ],
                [call(AuthService.storeAccessToken, exchangedToken), dynamic(storeAccessTokenMock)],
                [call(AuthService.getAndClearRequestedLocation), requestedLocation],
                [call(AuthService.logout), null],
                [call(fetchUserRolePermissions), { response: {} }],
                [call(AuthService.fetchAvailableProviderTypes), { response: [] }],
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

    it('should handle OIDC response with authorize roxctl mode', () => {
        delete window.location;
        window.location = { assign: jest.fn() };
        const token = 'my-token';
        const requestedLocation = '/my-location';
        const storeAccessTokenMock = jest.fn();
        const callbackURL = 'http://localhost:8080/';
        const serverState = `provider-id:2ed17ca6-4b3c-4279-8317-f26f8ba01c52#${callbackURL}`;

        return expectSaga(saga)
            .provide([
                ...createStateSelectors(),
                [call(AuthService.fetchLoginAuthProviders), { response: [] }],
                [
                    call(AuthService.exchangeAuthToken, `#id_token=${token}`, 'oidc', serverState),
                    { token, clientState: callbackURL },
                ],
                [call(AuthService.storeAccessToken, token), dynamic(storeAccessTokenMock)],
                [call(AuthService.getAndClearRequestedLocation), requestedLocation],
                [call(AuthService.logout), null],
                [call(fetchUserRolePermissions), { response: {} }],
                [call(AuthService.fetchAvailableProviderTypes), { response: [] }],
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
                expect(window.location.assign).toHaveBeenCalledWith(
                    `${callbackURL}?error=&errorDescription=&token=${token}`
                );
            });
    });

    it('should handle SAML response with test mode', () => {
        const storeLocationMock = jest.fn();
        const user =
            'eyJ1c2VySWQiOiJ0ZXN0QHN0YWNrcm94LmNvbSIsImV4cGlyZXMiOiIwMDAxLTAxLTAxVDAwOjAwOjAwWiIsImF1dGhQcm92aWRlciI6eyJpZCI6ImRlZjQzMDdjLTczMmEtNDUzZS05NzAyLTE2ZDU3NjA5MGE1NCIsIm5hbWUiOiJWYW5TYW1sT2t0YTEiLCJ0eXBlIjoic2FtbCIsInVpRW5kcG9pbnQiOiJsb2NhbGhvc3Q6ODAwMCIsImVuYWJsZWQiOnRydWUsImxvZ2luVXJsIjoiL3Nzby9sb2dpbi9kZWY0MzA3Yy03MzJhLTQ1M2UtOTcwMi0xNmQ1NzYwOTBhNTQifSwidXNlckluZm8iOnsidXNlcm5hbWUiOiJ0ZXN0QHN0YWNrcm94LmNvbSIsInBlcm1pc3Npb25zIjp7Im5hbWUiOiJBZG1pbiIsImdsb2JhbEFjY2VzcyI6IlJFQURfV1JJVEVfQUNDRVNTIn0sInJvbGVzIjpbeyJuYW1lIjoiQWRtaW4iLCJnbG9iYWxBY2Nlc3MiOiJSRUFEX1dSSVRFX0FDQ0VTUyJ9XX0sInVzZXJBdHRyaWJ1dGVzIjpbeyJrZXkiOiJlbWFpbCIsInZhbHVlcyI6WyJqd0BzdGFja3JveC5jb20iXX0seyJrZXkiOiJ1c2VyaWQiLCJ2YWx1ZXMiOlsidGVzdEBzdGFja3JveC5jb20iXX1dfQ';
        const requestedLocation = '/test-login-results';
        // TODO: mock auth action call, too
        // const setAuthProviderTestResultsMock = jest.fn();

        return expectSaga(saga)
            .provide([
                ...createStateSelectors(),
                [call(AuthService.fetchLoginAuthProviders), { response: [] }],
                // TODO: mock auth action call, too
                // [
                //     call(actions.setAuthProviderTestResults, {}),
                //     dynamic(setAuthProviderTestResultsMock),
                // ],
                [call(AuthService.getAndClearRequestedLocation), requestedLocation],
                [
                    call(AuthService.storeRequestedLocation, requestedLocation),
                    dynamic(storeLocationMock),
                ],
                [call(AuthService.logout), null],
                [call(fetchUserRolePermissions), { response: {} }],
                [call(AuthService.fetchAvailableProviderTypes), { response: [] }],
            ])
            .put(push(requestedLocation))
            .dispatch(
                createLocationChange(
                    '/auth/response/generic',
                    null,
                    `#state=&test=true&type=saml&user=${user}`
                )
            )
            .silentRun()
            .then(() => {
                // TODO: mock auth action call, too
                // expect(setAuthProviderTestResultsMock.mock.calls.length).toBe(1);
                expect(storeLocationMock.mock.calls.length).toBe(1);
            });
    });

    it('should logout in case of 401 HTTP error', () =>
        expectSaga(saga)
            .provide([
                ...createStateSelectors([{ name: 'ap1' }], AUTH_STATUS.LOGGED_IN),
                [call(AuthService.fetchLoginAuthProviders), { response: [{ name: 'ap1' }] }],
                [call(AuthService.getAccessToken), 'my-token'],
                [call(AuthService.logout), null],
                [call(fetchUserRolePermissions), { response: {} }],
                [call(AuthService.fetchAvailableProviderTypes), { response: [] }],
            ])
            .put(actions.logout())
            .dispatch(createLocationChange('/'))
            .dispatch(actions.handleAuthHttpError(new AuthService.AuthHttpError('error', 401)))
            .silentRun());

    it('should ignore 403 HTTP error', () =>
        expectSaga(saga)
            .provide([
                ...createStateSelectors([{ name: 'ap1' }], AUTH_STATUS.LOGGED_IN),
                [call(AuthService.fetchLoginAuthProviders), { response: [{ name: 'ap1' }] }],
                [call(AuthService.getAccessToken), 'my-token'],
                [call(AuthService.logout), null],
                [call(fetchUserRolePermissions), { response: {} }],
                [call(AuthService.fetchAvailableProviderTypes), { response: [] }],
            ])
            .not.put(actions.logout())
            .dispatch(createLocationChange('/'))
            .dispatch(actions.handleAuthHttpError(new AuthService.AuthHttpError('error', 403)))
            .silentRun());
});
