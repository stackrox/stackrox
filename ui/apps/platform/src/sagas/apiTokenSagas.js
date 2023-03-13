import { all, take, takeLatest, call, fork, put } from 'redux-saga/effects';

import { integrationsPath } from 'routePaths';
import * as service from 'services/APITokensService';
import { actions, types } from 'reducers/apitokens';
import { actions as notificationActions } from 'reducers/notifications';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';

function* getAPITokens() {
    try {
        const result = yield call(service.fetchAPITokens);
        yield put(actions.fetchAPITokens.success(result.response));
    } catch (error) {
        yield put(actions.fetchAPITokens.failure(error));
    }
}

function* revokeAPITokens({ ids }) {
    try {
        yield call(service.revokeAPITokens, ids);
        yield fork(getAPITokens);
        yield put(
            notificationActions.addNotification(
                `Successfully revoked ${ids.length} token${ids.length > 1 ? 's' : ''}`
            )
        );
        yield put(notificationActions.removeOldestNotification());
    } catch (error) {
        if (error.response) {
            yield put(notificationActions.addNotification(error.response.data.error));
            yield put(notificationActions.removeOldestNotification());
        }
    }
}

function* watchLocation() {
    yield takeEveryNewlyMatchedLocation(integrationsPath, getAPITokens);
}

function* watchFetchRequest() {
    while (true) {
        yield take([types.FETCH_API_TOKENS.REQUEST]);
        yield fork(getAPITokens);
    }
}

function* watchRevokeRequest() {
    yield takeLatest(types.REVOKE_API_TOKENS, revokeAPITokens);
}

export default function* integrations() {
    yield all([fork(watchLocation), fork(watchFetchRequest), fork(watchRevokeRequest)]);
}
