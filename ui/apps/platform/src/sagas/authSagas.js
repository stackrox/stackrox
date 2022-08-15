import { all, take, call, fork, put, takeLatest, takeEvery, select } from 'redux-saga/effects';
import { delay } from 'redux-saga';
import { push } from 'connected-react-router';
import queryString from 'qs';
import Raven from 'raven-js';
import { Base64 } from 'js-base64';

import {
    loginPath,
    testLoginResultsPath,
    accessControlPath,
    authResponsePrefix,
    integrationsPath,
} from 'routePaths';
import { takeEveryLocation, takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';
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
    const token = yield call(AuthService.getAccessToken);
    const tokenExists = !!token;

    // No token but validated providers present? Log out the user since they
    // can't have access.
    if (!tokenExists && authStatus !== AUTH_STATUS.LOGGED_OUT) {
        // it can happen if user had ANONYMOUS access before, but now auth provider was added to the system
        yield put(actions.logout());
        return;
    }

    // We have a token and some auth providers exist? Need to login if possible,
    // this will cause one of our providers to be authenticated, or, failing
    // that, remove the token since it is worthless.
    if (tokenExists && authStatus !== AUTH_STATUS.LOGGED_IN) {
        // typical situation if token was stored before and then auth providers were loaded
        try {
            const result = yield call(AuthService.getAuthStatus);
            // call didn't fail, meaning that the token is fine (should we check the returned result?)
            yield put(actions.login(result));
        } catch (e) {
            // call failed, assuming that the token is invalid
            yield put(actions.logout());
        }
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

function parseFragment(location) {
    const hash = queryString.parse(location.hash.slice(1)); // ignore '#' https://github.com/ljharb/qs/issues/222
    // The fragment as a whole is URL-encoded, which means that each individual field is doubly URL-encoded. We need
    // to decode one additional level of URL encoding here.
    const transformedHash = {};
    Object.entries(hash).forEach(([key, value]) => {
        transformedHash[key] = decodeURIComponent(value);
    });
    return transformedHash;
}

// isTestMode returns whether the given client-side state (of the general form
// `<auth provider ID>:<test prefix or empty>#<client state>`) indicates that we are in test mode).
// See `ParseClientState` in `pkg/auth/authproviders/idputil/state.go` for the authoritative implementation.
function isTestMode(state) {
    const stateComponents = state?.split(':') || [];
    const origStateComponents = stateComponents[1]?.split('#') || [];
    return origStateComponents[0] === testLoginClientState;
}

function* handleOidcResponse(location) {
    const hash = parseFragment(location);
    if (hash.error) {
        return { ...hash, test: isTestMode(hash.state) };
    }

    try {
        const { state, ...otherFields } = hash;
        const pseudoToken = `#${queryString.stringify({ ...otherFields })}`;
        const result = yield call(AuthService.exchangeAuthToken, pseudoToken, 'oidc', state);
        return result;
    } catch (error) {
        if (error.response) {
            return { error: error.response.data.error };
        }
        return { error: 'Could not exchange OIDC ID token' };
    }
}

function handleGenericResponse(location) {
    const hash = parseFragment(location);
    if (hash.error || !hash?.token) {
        return hash;
    }
    return { token: hash.token };
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
    }

    // save the test response for the results page to display
    yield put(actions.setAuthProviderTestResults(parsedResult));

    // set up the redirect to the results page
    yield call(AuthService.storeRequestedLocation, testLoginResultsPath);
}

function* dispatchAuthResponse(type, location) {
    // For every handler registered under `/auth/response/<type>`, add a function that returns the token.
    const responseHandlers = {
        oidc: handleOidcResponse,
        generic: handleGenericResponse,
    };

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
    } else if (result?.token) {
        yield call(AuthService.storeAccessToken, result.token);

        // TODO-ivan: seems like react-router-redux doesn't like pushing an action synchronously while handling LOCATION_CHANGE,
        // the bug is that it doesn't produce LOCATION_CHANGE event for this next push. Waiting here should be ok for an user.
        yield delay(10);
    } else {
        yield call(handleErrAuthResponse, result, `no auth token received via method ${type}`);
    }

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
            if (!remaining.active && !(remaining.traits?.mutabilityMode !== 'ALLOW_MUTATE')) {
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

function* watchLocationForAuthProviders() {
    const effects = [accessControlPath, integrationsPath].map((path) =>
        takeEveryNewlyMatchedLocation(path, getAuthProviders)
    );
    yield all(effects);
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
        fork(watchLocationForAuthProviders),
        takeEveryLocation(loginPath, handleLoginPageRedirect),
        fork(watchSaveAuthProvider),
        fork(watchDeleteAuthProvider),
        fork(watchAuthProvidersFetchRequest),
        fork(watchLoginAuthProvidersFetchRequest),
        fork(watchLogout),
        fork(watchAuthHttpErrors),
    ]);
}
