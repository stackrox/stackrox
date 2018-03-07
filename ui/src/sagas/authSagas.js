import { all, take, call, fork, put } from 'redux-saga/effects';

import AuthService from 'services/AuthService';
import { actions, types } from 'reducers/auth';
import { types as locationActionTypes } from 'reducers/routes';

const integrationsPath = '/main/integrations';

export function* getAuthProviders() {
    try {
        const result = yield call(AuthService.updateAuthProviders);
        yield put(actions.fetchAuthProviders.success(result.response));
    } catch (error) {
        yield put(actions.fetchAuthProviders.failure(error));
    }
}

export function* watchIntegrationsLocation() {
    while (true) {
        const action = yield take(locationActionTypes.LOCATION_CHANGE);
        const { payload: location } = action;
        if (location && location.pathname && location.pathname.startsWith(integrationsPath)) {
            yield fork(getAuthProviders);
        }
    }
}

function* watchFetchRequest() {
    while (true) {
        yield take(types.FETCH_AUTH_PROVIDERS.REQUEST);
        yield fork(getAuthProviders);
    }
}

export default function* auth() {
    yield all([fork(watchIntegrationsLocation), fork(watchFetchRequest)]);
}
