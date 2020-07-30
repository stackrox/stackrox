import { call, fork, put, race, take } from 'redux-saga/effects';

import { fetchMetadata } from 'services/MetadataService';
import { actions, METADATA_LICENSE_STATUS } from 'reducers/metadata';
import { delay } from 'redux-saga';
import { types } from 'reducers/license';

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
        // eslint-disable-next-line
        const metadata = yield call(fetchVersionAndSendTo, action);
        if (metadata) {
            action = actions.pollMetadata;
        }
        const nextPoll =
            !metadata || metadata.licenseStatus === METADATA_LICENSE_STATUS.RESTARTING
                ? 1000
                : 10000;
        yield race([call(delay, nextPoll), take(types.SET_LICENSE_UPLOAD_STATUS)]);
    }
}

export function* pollUntilCentralRestarts() {
    const action = actions.initialFetchMetadata;
    let continuePolling = true;
    while (continuePolling) {
        try {
            const result = yield call(fetchVersionAndSendTo, action);
            const { licenseStatus } = result;
            if (licenseStatus !== METADATA_LICENSE_STATUS.RESTARTING) {
                continuePolling = false;
            }
        } catch (error) {
            continuePolling = false;
        }
        delay(1000);
    }
}

export default function* metadata() {
    yield fork(pollVersion);
}
