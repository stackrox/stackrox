import { all, call, fork, put } from 'redux-saga/effects';

import { fetchMetadata } from 'services/MetadataService';
import { actions } from 'reducers/metadata';
import { delay } from 'redux-saga';

// Fetches the version and sends it to the given action.
// The action must be a "fetching" action type, which has
// .success and .failure methods.
function* fetchVersionAndSendTo(action) {
    try {
        const result = yield call(fetchMetadata);
        yield put(action.success(result.response));
    } catch (error) {
        yield put(action.failure(error));
    }
}

function* fetchVersion() {
    yield call(fetchVersionAndSendTo, actions.initialFetchMetadata);
}

function* pollVersion() {
    while (true) {
        yield call(delay, 10000);
        yield call(fetchVersionAndSendTo, actions.pollMetadata);
    }
}

export default function* metadata() {
    yield all([fork(fetchVersion), fork(pollVersion)]);
}
