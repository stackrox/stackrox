import { all, call, fork, put, take } from 'redux-saga/effects';

import { integrationsPath } from 'routePaths';
import { fetchAPITokens as serviceFetchAPITokens } from 'services/APITokensService';
import { actions, types } from 'reducers/apitokens';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';

function* getAPITokens() {
    try {
        const result = yield call(serviceFetchAPITokens);
        yield put(actions.fetchAPITokens.success(result.response));
    } catch (error) {
        yield put(actions.fetchAPITokens.failure(error));
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

export default function* integrations() {
    yield all([fork(watchLocation), fork(watchFetchRequest)]);
}
