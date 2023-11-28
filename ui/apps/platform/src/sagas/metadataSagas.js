import { call, fork, put } from 'redux-saga/effects';

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
        return result.response;
    } catch (error) {
        yield put(action.failure(error));
    }
    return null;
}

function* pollVersion() {
    let action = actions.initialFetchMetadata;
    while (true) {
        const metadata = yield call(fetchVersionAndSendTo, action);
        if (metadata && metadata.version) {
            action = actions.pollMetadata;
        }
        const nextPoll = !metadata ? 1000 : 10000;
        yield call(delay, nextPoll);
    }
}

export default function* metadata() {
    yield fork(pollVersion);
}
