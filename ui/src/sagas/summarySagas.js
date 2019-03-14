import { take, call, fork, put, cancel } from 'redux-saga/effects';
import { delay } from 'redux-saga';

import { mainPath } from 'routePaths';
import fetchSummaryCounts from 'services/SummaryService';
import { actions } from 'reducers/summaries';
import { types as locationActionTypes } from 'reducers/routes';

export function* pollSummaryCounts() {
    while (true) {
        try {
            const result = yield call(fetchSummaryCounts);
            yield put(actions.fetchSummaryCounts.success(result.response));
        } catch (error) {
            yield put(actions.fetchSummaryCounts.failure(error));
        }
        yield call(delay, 30000); // poll every 30 sec
    }
}

export function* watchLocation() {
    let pollTask = null;
    while (true) {
        const action = yield take(locationActionTypes.LOCATION_CHANGE);
        const { payload: location } = action;

        if (location && location.pathname && location.pathname.startsWith(mainPath)) {
            // start only if it's not already in progress
            if (!pollTask) {
                pollTask = yield fork(pollSummaryCounts);
            }
        } else if (pollTask) {
            yield cancel(pollTask);
            pollTask = null;
        }
    }
}

export default function* summaries() {
    yield fork(watchLocation);
}
