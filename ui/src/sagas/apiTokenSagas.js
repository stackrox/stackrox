import { all, take, takeLatest, call, fork, put, select } from 'redux-saga/effects';

import { integrationsPath } from 'routePaths';
import * as service from 'services/APITokensService';
import { actions, types, apiTokenFormId } from 'reducers/apitokens';
import { actions as roleActions } from 'reducers/roles';
import { actions as notificationActions } from 'reducers/notifications';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';
import { getFormValues } from 'redux-form';

function* getAPITokens() {
    try {
        const result = yield call(service.fetchAPITokens);
        yield put(actions.fetchAPITokens.success(result.response));
    } catch (error) {
        yield put(actions.fetchAPITokens.failure(error));
    }
}

function* generateAPIToken() {
    try {
        const formData = yield select(getFormValues(apiTokenFormId));
        const result = yield call(service.generateAPIToken, formData);
        yield put(actions.generateAPIToken.success(result.response));
    } catch (error) {
        if (error.response) {
            yield put(notificationActions.addNotification(error.response.data.error));
            yield put(notificationActions.removeOldestNotification());
        }
        yield put(actions.generateAPIToken.failure(error));
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
        yield take([types.FETCH_API_TOKENS.REQUEST, types.GENERATE_API_TOKEN.SUCCESS]);
        yield fork(getAPITokens);
    }
}

function* watchGenerateRequest() {
    yield takeLatest(types.GENERATE_API_TOKEN.REQUEST, generateAPIToken);
}

function* watchRevokeRequest() {
    yield takeLatest(types.REVOKE_API_TOKENS, revokeAPITokens);
}

function* requestFetchRoles() {
    yield put(roleActions.fetchRoles.request());
}

function* watchModalOpen() {
    yield takeLatest(types.START_TOKEN_GENERATION_WIZARD, requestFetchRoles);
}

export default function* integrations() {
    yield all([
        fork(watchLocation),
        fork(watchFetchRequest),
        fork(watchGenerateRequest),
        fork(watchRevokeRequest),
        fork(watchModalOpen)
    ]);
}
