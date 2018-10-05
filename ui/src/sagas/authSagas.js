import { all, take, call, fork, put, takeLatest, takeEvery, select } from 'redux-saga/effects';
import { delay } from 'redux-saga';
import { push } from 'react-router-redux';
import queryString from 'query-string';

import { loginPath, integrationsPath, authResponsePrefix } from 'routePaths';
import { takeEveryLocation } from 'utils/sagaEffects';
import * as AuthService from 'services/AuthService';
import { selectors } from 'reducers';
import { actions, types, AUTH_STATUS } from 'reducers/auth';
import { types as locationActionTypes } from 'reducers/routes';

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

function handleOidcResponse(location) {
    const hash = queryString.parse(location.hash);
    if (hash.error) {
        return hash;
    }
    return { token: hash.access_token };
}

function* dispatchAuthResponse(type, location) {
    // For every handler registered under `/auth/response/<type>`, add a function that returns the token.
    const responseHandlers = {
        oidc: handleOidcResponse
    };
    let result = {};
    const handler = responseHandlers[type];
    if (handler) {
        result = handler(location);
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

function* watchAuthHttpErrors() {
    yield takeEvery(types.AUTH_HTTP_ERROR, handleHttpError);
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
        fork(watchAuthProvidersFetchRequest),
        fork(watchLogout),
        fork(watchAuthHttpErrors)
    ]);
}
