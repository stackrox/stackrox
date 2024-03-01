import { all, take, call, fork, put, takeLatest, takeEvery, select } from 'redux-saga/effects';
import { delay } from 'redux-saga';
import { push } from 'connected-react-router';
import queryString from 'qs';
import Raven from 'raven-js';
import { Base64 } from 'js-base64';

import { loginPath, testLoginResultsPath, authResponsePrefix } from 'routePaths';
import { takeEveryLocation } from 'utils/sagaEffects';
import { parseAndDecodeFragment } from 'utils/parseAndDecodeFragment';
import * as AuthService from 'services/AuthService';
import fetchUsersAttributes from 'services/AttributesService';
import { fetchUserRolePermissions } from 'services/RolesService';
import { selectors } from 'reducers';
import { actions, types, AUTH_STATUS } from 'reducers/auth';
import { actions as groupActions } from 'reducers/groups';
import { types as locationActionTypes } from 'reducers/routes';
import { actions as notificationActions } from 'reducers/notifications';
import { actions as rolesActions } from 'reducers/roles';

// The unique string indicating auth provider test mode. Do not change!
// Must be kept in sync with `TestLoginClientState` in `pkg/auth/authproviders/idputil/state.go`.
const testLoginClientState = `e003ba41-9cc1-48ee-b6a9-2dd7c21da92e`;
// The unique string indicating auth provider authorize roxctl mode. Do not change!
// Must be kept in sync with `AuthorizeRoxctlClientState` in `pkg/auth/authproviders/idputil/state.go`.
const authorizeRoxctlClientState = `2ed17ca6-4b3c-4279-8317-f26f8ba01c52`;

function* getUserPermissions() {
    /*
     * Call request because userRolePermissions.isLoading reducer needs the action
     * for subsequent requests (for example, manual refresh; or log out, and then log in again).
     * Imitate request-success-failure pattern in redux-thunk.
     * In this case, redux-saga makes the request independently of the action.
     */
    yield put(rolesActions.fetchUserRolePermissions.request());
    try {
        const result = yield call(fetchUserRolePermissions);
        yield put(rolesActions.fetchUserRolePermissions.success(result.response));
    } catch (error) {
        yield put(rolesActions.fetchUserRolePermissions.failure(error));
    }
}

function* evaluateUserAccess() {
    const authStatus = yield select(selectors.getAuthStatus);

    // Previously, we eagerly tried a logout / login based on whether the token exists within our state.
    // However, since we now do not know this due to the cookie being a HTTP-only one, we will not explicitly
    // call logout for unauthorized auth status.
    if (authStatus === AUTH_STATUS.LOGGED_OUT) {
        return;
    }

    try {
        const result = yield call(AuthService.getAuthStatus);
        // call didn't fail, meaning that the token is fine (should we check the returned result?)
        yield put(actions.login(result));
    } catch (e) {
        yield put(actions.logout());
    }
}

function* watchNewAuthProviders() {
    yield takeLatest(types.FETCH_LOGIN_AUTH_PROVIDERS.SUCCESS, evaluateUserAccess);
}

export function* getLoginAuthProviders() {
    try {
        const result = yield call(AuthService.fetchLoginAuthProviders);
        yield put(actions.fetchLoginAuthProviders.success(result?.response || []));
    } catch (error) {
        yield put(actions.fetchLoginAuthProviders.failure(error));
    }
}

export function* getAuthProviders() {
    try {
        const result = yield call(AuthService.fetchAuthProviders);
        yield put(actions.fetchAuthProviders.success(result?.response || []));
    } catch (error) {
        yield put(actions.fetchAuthProviders.failure(error));
    }
}

function* watchAuthProvidersFetchRequest() {
    yield takeLatest(types.FETCH_AUTH_PROVIDERS.REQUEST, getAuthProviders);
}

function* watchLoginAuthProvidersFetchRequest() {
    yield takeLatest(types.FETCH_LOGIN_AUTH_PROVIDERS.REQUEST, getLoginAuthProviders);
}

function* logout() {
    yield call(AuthService.logout);
}

function* watchLogout() {
    yield takeLatest(types.LOGOUT, logout);
}

function* handleLoginPageRedirect({ location }) {
    const { state } = location;
    if (state && state.from && !state.from.startsWith(loginPath)) {
        // we were redirected to login page from another page
        yield call(AuthService.storeRequestedLocation, state.from);
    }
}

// isTestMode returns whether the given client-side state (of the general form
// `<auth provider ID>:<test prefix or empty>#<client state>`) indicates that we are in test mode.
// See `ParseClientState` in `pkg/auth/authproviders/idputil/state.go` for the authoritative implementation.
function isTestMode(state) {
    return isGivenMode(state, testLoginClientState);
}

// isAuthorizeRoxctlMode returns whether the given client-side state (of the general form
// `<auth provider ID>:<authorize roxctl state or empty>#<client state>`) indicates that we are in authorize
// roxctl mode.
// See `ParseClientState` in `pkg/auth/authproviders/idputil/state.go` for the authoritative implementation.
function isAuthorizeRoxctlMode(state) {
    return isGivenMode(state, authorizeRoxctlClientState);
}

function isGivenMode(state, mode) {
    const stateComponents = state?.split(':') || [];
    const origStateComponents = stateComponents[1]?.split('#') || [];
    return origStateComponents[0] === mode;
}

function* handleOidcResponse(location) {
    const parsedFragment = parseAndDecodeFragment(location);
    if (parsedFragment.has('error')) {
        const state = parsedFragment.get('state');
        parsedFragment.set('test', isTestMode(state).toString());
        parsedFragment.set('authorizeRoxctl', isAuthorizeRoxctlMode(state).toString());
        return Object.fromEntries(parsedFragment.entries());
    }

    try {
        const state = parsedFragment.get('state');
        const otherFields = Object.fromEntries(
            Array.from(parsedFragment.entries()).filter(([key]) => key !== 'state')
        );
        const pseudoToken = `#${queryString.stringify({ ...otherFields })}`;
        const result = yield call(AuthService.exchangeAuthToken, pseudoToken, 'oidc', state);
        result.authorizeRoxctl = isAuthorizeRoxctlMode(state);
        return result;
    } catch (error) {
        if (error.response) {
            return { error: error.response.data.error };
        }
        return { error: 'Could not exchange OIDC ID token' };
    }
}

function handleGenericResponse(location) {
    const parsedFragment = parseAndDecodeFragment(location);
    if (parsedFragment.has('error') || !parsedFragment.has('token')) {
        return Object.fromEntries(parsedFragment.entries());
    }
    return {
        token: parsedFragment.get('token'),
        authorizeRoxctl: isAuthorizeRoxctlMode(parsedFragment.get('state')),
    };
}

function* handleErrAuthResponse(result, defaultErrMsg) {
    if (!result?.error) {
        yield put(actions.handleIdpError({ error: defaultErrMsg }));
    }
    yield put(actions.handleIdpError(result));
}

function* handleTestLoginAuthResponse(location, type, result) {
    const parsedResult = {
        error: result?.error || null,
        error_description: result?.error_description || null,
    };

    if (result?.user) {
        let user = {};
        try {
            user = JSON.parse(Base64.decode(result.user)); // built-in atob not URL or UTF safe
        } catch (error) {
            // not base64 encoded
            user = result?.user;
        }
        parsedResult.userID = user.userId || null;
        parsedResult.userAttributes = user.userAttributes || null;
        parsedResult.roles = user.userInfo?.roles || [];
        parsedResult.idpToken = user.idpToken || null;
    }

    // save the test response for the results page to display
    yield put(actions.setAuthProviderTestResults(parsedResult));

    // set up the redirect to the results page
    yield call(AuthService.storeRequestedLocation, testLoginResultsPath);
}

function* handleAuthorizeRoxctlLoginResponse(result) {
    const query = {
        error: result?.error || null,
        errorDescription: result?.error_description || null,
        token: result?.token || null,
    };
    // Verify that the callback URL is pointing to localhost.
    const parsedCallbackURL = new URL(result.clientState);
    if (parsedCallbackURL.hostname !== 'localhost' && parsedCallbackURL.hostname !== '127.0.0.1') {
        yield call(
            handleErrAuthResponse,
            result,
            'Invalid callback URL given for roxctl authorization. Only localhost is allowed as callback'
        );
    }

    // Redirect to the callback URL (i.e. the server opened by roxctl central login) with the token as query parameter
    // or any error that may have occurred.
    window.location.assign(`${parsedCallbackURL.toString()}?${queryString.stringify(query)}`);
}

function* dispatchAuthResponse(type, location) {
    // For every handler registered under `/auth/response/<type>`, add a function that returns the token.
    const responseHandlers = {
        oidc: handleOidcResponse,
        generic: handleGenericResponse,
    };
    yield call(AuthService.dispatchResponseStarted);

    let result = {};
    const handler = responseHandlers[type];
    if (handler) {
        result = yield call(handler, location);
    } else {
        yield call(handleErrAuthResponse, result, `unknown auth response type ${type}`);
    }

    if (result?.test === true || result?.test === 'true') {
        // `test` property can be a string or boolean, depending on the type of provider
        //    but if it is present in any form, its a test of the provider and not an actual login
        yield call(handleTestLoginAuthResponse, location, type, result);
    } else if (result?.authorizeRoxctl === true || result?.authorizeRoxctl === 'true') {
        yield call(handleAuthorizeRoxctlLoginResponse, result);
    } else if (result?.token) {
        // TODO-ivan: seems like react-router-redux doesn't like pushing an action synchronously while handling LOCATION_CHANGE,
        // the bug is that it doesn't produce LOCATION_CHANGE event for this next push. Waiting here should be ok for an user.
        yield delay(10);
    } else {
        yield call(handleErrAuthResponse, result, `no auth token received via method ${type}`);
    }

    yield call(AuthService.dispatchResponseFinished);

    yield fork(getUserPermissions);

    const storedLocation = yield call(AuthService.getAndClearRequestedLocation);
    yield put(push(storedLocation || '/')); // try to restore requested path
    yield call(getLoginAuthProviders);
}

function* handleHttpError(action) {
    const { error } = action;
    if (error.isAccessDenied()) {
        // TODO-ivan: for now leave it to individual calls to deal with (e.g. popup message etc.)
    } else {
        // access was revoked or auth mode was enabled, need to update auth providers
        yield fork(getLoginAuthProviders);
        yield put(actions.logout());
    }
}

function* saveAuthProvider(action) {
    try {
        const { authProvider } = action;
        const { groups, defaultRole, ...remaining } = authProvider;
        const authProviders = yield select(selectors.getAuthProviders);
        const filteredGroups = groups.filter(
            (group) =>
                group &&
                group.props &&
                group.props.key &&
                group.props.key !== '' &&
                group.roleName &&
                group.roleName !== ''
        );

        yield put(
            actions.setSaveAuthProviderStatus({
                status: 'saving',
                message: '',
            })
        );

        const isNewAuthProvider = !authProviders.filter(
            (currAuthProvider) => currAuthProvider.name === remaining.name
        ).length;
        if (isNewAuthProvider) {
            const savedAuthProvider = yield call(AuthService.saveAuthProvider, remaining);
            filteredGroups.forEach((group) =>
                Object.assign(group.props, { authProviderId: savedAuthProvider.data.id })
            );
            yield put(
                groupActions.saveRuleGroup(filteredGroups, defaultRole, savedAuthProvider.data.id)
            );
            yield call(getAuthProviders);
            yield call(fetchUsersAttributes);
            yield put(actions.selectAuthProvider({ ...remaining, id: savedAuthProvider.data.id }));
        } else {
            const isImmutable = yield call(AuthService.getIsAuthProviderImmutable, remaining);
            if (!remaining.active && !isImmutable) {
                yield call(AuthService.saveAuthProvider, remaining);
            }
            yield call(getAuthProviders);
            yield call(fetchUsersAttributes);
            yield put(groupActions.saveRuleGroup(filteredGroups, defaultRole, authProvider.id));
            const newAuthProviders = yield select(selectors.getAuthProviders);
            const updatedSelectedAuthProvider = newAuthProviders.find(
                (provider) => provider.id === authProvider.id
            );
            yield put(actions.selectAuthProvider(updatedSelectedAuthProvider));
        }
    } catch (error) {
        yield put(actions.setAuthProviderEditingState(true));
        const message =
            (error.response && error.response.data && error.response.data.error) ||
            'AuthProvider request timed out';
        yield put(
            actions.setSaveAuthProviderStatus({
                status: 'error',
                message,
            })
        );
        Raven.captureException(error);
    }
}

function* deleteAuthProvider(action) {
    const { id } = action;
    try {
        yield call(AuthService.deleteAuthProvider, id);
        yield put(actions.fetchAuthProviders.request());
    } catch (error) {
        yield put(
            notificationActions.addNotification(
                (error.response && error.response.data && error.response.data.error) ||
                    'AuthProvider request timed out'
            )
        );
        yield put(notificationActions.removeOldestNotification());
        Raven.captureException(error);
    }
}

function* watchAuthHttpErrors() {
    yield takeEvery(types.AUTH_HTTP_ERROR, handleHttpError);
}

function* watchSaveAuthProvider() {
    yield takeLatest(types.SAVE_AUTH_PROVIDER, saveAuthProvider);
}

function* watchDeleteAuthProvider() {
    yield takeLatest(types.DELETE_AUTH_PROVIDER, deleteAuthProvider);
}

function* fetchAvailableProviderTypes() {
    try {
        const result = yield call(AuthService.fetchAvailableProviderTypes);
        yield put(actions.setAvailableProviderTypes(result?.response || []));
    } catch (error) {
        yield put(
            notificationActions.addNotification(
                (error.response && error.response.data && error.response.data.error) ||
                    'AuthProvider Types request timed out'
            )
        );
        yield put(notificationActions.removeOldestNotification());
        Raven.captureException(error);
    }
}

export default function* auth() {
    // start by monitoring auth providers to re-evaluate user access
    yield fork(watchNewAuthProviders);
    yield fork(fetchAvailableProviderTypes);

    // take the first location change, i.e. the location where user landed first time
    const action = yield take(locationActionTypes.LOCATION_CHANGE);
    const {
        payload: { location },
    } = action;
    if (location.pathname?.startsWith(authResponsePrefix)) {
        // if it was a redirect after authentication, handle it properly
        const authType = location.pathname.substr(authResponsePrefix.length);
        yield fork(dispatchAuthResponse, authType, location);
    } else {
        // otherwise we still need to fetch auth providers to check if user can access the app
        yield fork(getLoginAuthProviders);
        yield fork(getUserPermissions);
    }

    yield all([
        takeEveryLocation(loginPath, handleLoginPageRedirect),
        fork(watchSaveAuthProvider),
        fork(watchDeleteAuthProvider),
        fork(watchAuthProvidersFetchRequest),
        fork(watchLoginAuthProvidersFetchRequest),
        fork(watchLogout),
        fork(watchAuthHttpErrors),
    ]);
}
