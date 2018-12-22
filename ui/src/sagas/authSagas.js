import { all, take, call, fork, put, takeLatest, takeEvery, select } from 'redux-saga/effects';
import { delay } from 'redux-saga';
import { push } from 'react-router-redux';
import queryString from 'query-string';
import Raven from 'raven-js';

import { loginPath, integrationsPath, authResponsePrefix } from 'routePaths';
import { takeEveryLocation } from 'utils/sagaEffects';
import * as AuthService from 'services/AuthService';
import fetchUsersAttributes from 'services/AttributesService';
import { selectors } from 'reducers';
import { actions, types, AUTH_STATUS } from 'reducers/auth';
import { actions as groupActions } from 'reducers/groups';
import { types as locationActionTypes } from 'reducers/routes';
import { actions as notificationActions } from 'reducers/notifications';

function* evaluateUserAccess() {
    const authProviders = yield select(selectors.getAuthProviders);
    const authStatus = yield select(selectors.getAuthStatus);
    const tokenExists = yield call(AuthService.isTokenPresent);

    // No providers? Or no token and providers that aren't validated?
    // Just grant anonymous access.
    const validatedProviders = authProviders.filter(pro => pro.validated);
    if (authProviders.length === 0 || (!tokenExists && validatedProviders.length === 0)) {
        if (authStatus !== AUTH_STATUS.ANONYMOUS_ACCESS) {
            // whatever auth status was before, no auth providers mean anonymous access is allowed
            yield call(AuthService.clearAccessToken);
            yield put(actions.grantAnonymousAccess());
        }
        return;
    }

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
            yield call(AuthService.fetchAuthStatus);
            // call didn't fail, meaning that the token is fine (should we check the returned result?)
            yield put(actions.login());
        } catch (e) {
            // call failed, assuming that the token is invalid
            yield put(actions.logout());
        }
    }
}

function* watchNewAuthProviders() {
    yield takeLatest(types.FETCH_AUTH_PROVIDERS.SUCCESS, evaluateUserAccess);
}

export function* getAuthProviders() {
    try {
        const result = yield call(AuthService.fetchAuthProviders);
        yield put(actions.fetchAuthProviders.success(result.response));
    } catch (error) {
        yield put(actions.fetchAuthProviders.failure(error));
    }
}

function* watchAuthProvidersFetchRequest() {
    yield takeLatest(types.FETCH_AUTH_PROVIDERS.REQUEST, getAuthProviders);
}

function* logout() {
    yield call(AuthService.clearAccessToken);
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

function* handleOidcResponse(location) {
    const hash = queryString.parse(location.hash);
    if (hash.error) {
        return hash;
    }
    try {
        const { id_token: idToken, state } = hash;
        const result = yield call(AuthService.exchangeAuthToken, idToken, 'oidc', state);
        return result;
    } catch (error) {
        if (error.response) {
            return { error: error.response.data.error };
        }
        return { error: 'Could not exchange OIDC ID token' };
    }
}

function handleGenericResponse(location) {
    const hash = queryString.parse(location.hash);
    if (hash.error) {
        return hash;
    }
    return { token: hash.token };
}

function* dispatchAuthResponse(type, location) {
    // For every handler registered under `/auth/response/<type>`, add a function that returns the token.
    const responseHandlers = {
        oidc: handleOidcResponse,
        generic: handleGenericResponse
    };
    let result = {};
    const handler = responseHandlers[type];
    if (handler) {
        result = yield call(handler, location);
    } else {
        result = { error: `unknown auth response type ${type}` };
    }

    if (result.token) {
        yield call(AuthService.storeAccessToken, result.token);

        // TODO-ivan: seems like react-router-redux doesn't like pushing an action synchronously while handling LOCATION_CHANGE,
        // the bug is that it doesn't produce LOCATION_CHANGE event for this next push. Waiting here should be ok for an user.
        yield delay(10);
    } else {
        if (!result || !result.error) {
            result = { error: `no auth token received via method ${type}` };
        }
        yield put(actions.handleIdpError(result));
    }

    const storedLocation = yield call(AuthService.getAndClearRequestedLocation);
    yield put(push(storedLocation || '/')); // try to restore requested path
    yield call(getAuthProviders);
}

function* handleHttpError(action) {
    const { error } = action;
    if (error.isAccessDenied()) {
        // TODO-ivan: for now leave it to individual calls to deal with (e.g. popup message etc.)
    } else {
        // access was revoked or auth mode was enabled, need to update auth providers
        yield fork(getAuthProviders);
        yield put(actions.logout());
    }
}

function* saveAuthProvider(action) {
    try {
        const { authProvider } = action;
        const { groups, defaultRole, ...remaining } = authProvider;
        const authProviders = yield select(selectors.getAuthProviders);
        const filteredGroups = groups.filter(
            group =>
                group &&
                group.props &&
                group.props.key &&
                group.props.key !== '' &&
                group.roleName &&
                group.roleName !== ''
        );
        const isNewAuthProvider = !authProviders.filter(
            currAuthProvider => currAuthProvider.name === remaining.name
        ).length;
        if (isNewAuthProvider) {
            const savedAuthProvider = yield call(AuthService.saveAuthProvider, remaining);
            filteredGroups.forEach(group =>
                Object.assign(group.props, { authProviderId: savedAuthProvider.data.id })
            );
            yield put(
                groupActions.saveRuleGroup(filteredGroups, defaultRole, savedAuthProvider.data.id)
            );
            yield call(getAuthProviders);
            yield call(fetchUsersAttributes);
            yield put(actions.selectAuthProvider({ ...remaining, id: savedAuthProvider.data.id }));
        } else {
            yield call(AuthService.saveAuthProvider, remaining);
            yield call(getAuthProviders);
            yield call(fetchUsersAttributes);
            yield put(groupActions.saveRuleGroup(filteredGroups, defaultRole));
            yield put(actions.selectAuthProvider(remaining));
        }
    } catch (error) {
        yield put(notificationActions.addNotification(error.response.data.error));
        yield put(notificationActions.removeOldestNotification());
        Raven.captureException(error);
    }
}

function* deleteAuthProvider(action) {
    const { id } = action;
    try {
        yield call(AuthService.deleteAuthProvider, id);
        yield put(actions.fetchAuthProviders.request());
    } catch (error) {
        yield put(notificationActions.addNotification(error.response.data.error));
        yield put(notificationActions.removeOldestNotification());
        Raven.captureException(error);
    }
}

function* selectAuthProvider(action) {
    const { authProvider } = action;
    try {
        if (!authProvider) {
            const authProviders = yield select(selectors.getAuthProviders);
            yield put(actions.selectAuthProvider(authProviders[0]));
        }
    } catch (error) {
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

function* watchSelectAuthProvider() {
    yield takeLatest(types.SELECTED_AUTH_PROVIDER, selectAuthProvider);
}

export default function* auth() {
    // start by monitoring auth providers to re-evaluate user access
    yield fork(watchNewAuthProviders);

    // take the first location change, i.e. the location where user landed first time
    const action = yield take(locationActionTypes.LOCATION_CHANGE);
    const { payload: location } = action;
    if (location.pathname && location.pathname.startsWith(authResponsePrefix)) {
        // if it was a redirect after authentication, handle it properly
        const authType = location.pathname.substr(authResponsePrefix.length);
        yield fork(dispatchAuthResponse, authType, location);
    } else {
        // otherwise we still need to fetch auth providers to check if user can access the app
        yield fork(getAuthProviders);
    }

    yield all([
        takeEveryLocation(integrationsPath, getAuthProviders),
        takeEveryLocation(loginPath, handleLoginPageRedirect),
        fork(watchSaveAuthProvider),
        fork(watchDeleteAuthProvider),
        fork(watchAuthProvidersFetchRequest),
        fork(watchLogout),
        fork(watchAuthHttpErrors),
        fork(watchSelectAuthProvider)
    ]);
}
