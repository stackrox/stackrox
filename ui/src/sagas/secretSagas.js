import { all, call, fork, put, select, takeLatest } from 'redux-saga/effects';

import { secretsPath } from 'routePaths';
import { fetchSecret, fetchSecrets } from 'services/SecretsService';
import { types, actions } from 'reducers/secrets';
import { takeEveryLocation, takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';
import { selectors } from 'reducers';

export function* getSecrets({ options = [] }) {
    try {
        const result = yield call(fetchSecrets, options);
        yield put(actions.fetchSecrets.success(result.response, { options }));
    } catch (error) {
        yield put(actions.fetchSecrets.failure(error));
    }
}

export function* getSecret(id) {
    try {
        const result = yield call(fetchSecret, id);
        yield put(actions.fetchSecret.success(result.response, { id }));
    } catch (error) {
        yield put(actions.fetchSecret.failure(error));
    }
}

function* filterSecretsPageBySearch() {
    const options = yield select(selectors.getSecretsSearchOptions);
    if (options.length && options[options.length - 1].type) {
        return;
    }
    yield fork(getSecrets, { options });
}

function* watchSecretSearchOptions() {
    yield takeLatest(types.SET_SEARCH_OPTIONS, filterSecretsPageBySearch);
}

function* getSelectedSecret({ match }) {
    const { secretId } = match.params;
    if (secretId) {
        yield put(actions.fetchSecret.request());
        yield call(getSecret, secretId);
    }
}

export default function* secrets() {
    yield all([
        takeEveryLocation(secretsPath, getSelectedSecret),
        takeEveryNewlyMatchedLocation(secretsPath, filterSecretsPageBySearch),
        fork(watchSecretSearchOptions)
    ]);
}
