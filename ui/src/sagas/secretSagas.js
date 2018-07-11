import { all, call, fork, put, select, takeLatest } from 'redux-saga/effects';

import { secretsPath } from 'routePaths';
import fetchSecrets from 'services/SecretsService';
import { types, actions } from 'reducers/secrets';
import { takeEveryLocation } from 'utils/sagaEffects';
import { selectors } from 'reducers';

export function* getSecrets({ options = [] }) {
    try {
        const result = yield call(fetchSecrets, options);
        yield put(actions.fetchSecrets.success(result.response, { options }));
    } catch (error) {
        yield put(actions.fetchSecrets.failure(error));
    }
}

function* filterSecretsPageBySearch() {
    const options = yield select(selectors.getSecretsSearchOptions);
    yield fork(getSecrets, { options });
}

function* watchSecretSearchOptions() {
    yield takeLatest(types.SET_SEARCH_OPTIONS, filterSecretsPageBySearch);
}

export default function* secrets() {
    yield all([
        takeEveryLocation(secretsPath, filterSecretsPageBySearch),
        fork(watchSecretSearchOptions)
    ]);
}
