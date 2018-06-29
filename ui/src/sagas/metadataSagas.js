import { call, fork, put } from 'redux-saga/effects';

import { fetchMetadata } from 'services/MetadataService';
import { actions } from 'reducers/metadata';

function* fetchVersion() {
    try {
        const result = yield call(fetchMetadata);
        yield put(actions.fetchMetadata.success(result.response));
    } catch (error) {
        yield put(actions.fetchMetadata.failure(error));
    }
}

export default function* metadata() {
    yield fork(fetchVersion);
}
