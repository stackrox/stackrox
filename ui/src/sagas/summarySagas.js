import { take, call, fork, put } from 'redux-saga/effects';
import { delay } from 'redux-saga';
import fetchSummaryCounts from 'services/SummaryService';
import { actions } from 'reducers/summaries';
import { types as locationActionTypes } from 'reducers/routes';

const mainPath = '/main/';

export function* pollSummaryCounts() {
    while (true) {
        try {
            const result = yield call(fetchSummaryCounts);
            yield put(actions.fetchSummaryCounts.success(result.response));
            yield call(delay, 5000); // poll every 5 sec
        } catch (error) {
            yield put(actions.fetchSummaryCounts.failure(error));
        }
    }
}

export function* watchLocation() {
    let shouldStartPolling = true;
    while (true) {
        const action = yield take(locationActionTypes.LOCATION_CHANGE);
        const { payload: location } = action;

        if (
            location &&
            location.pathname &&
            location.pathname.startsWith(mainPath) &&
            shouldStartPolling
        ) {
            yield fork(pollSummaryCounts);
            shouldStartPolling = false;
        }
    }
}

export default function* summaries() {
    yield fork(watchLocation);
}
