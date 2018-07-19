import { all, takeLatest, call, fork, put, select } from 'redux-saga/effects';

import { environmentPath } from 'routePaths';
import fetchNetworkGraph from 'services/EnvironmentService';
import { actions, types } from 'reducers/environment';
import { selectors } from 'reducers';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';

function* getNetworkGraph({ options = [] }) {
    try {
        const result = yield call(fetchNetworkGraph, options);
        yield put(actions.fetchNetworkGraph.success(result.response, { options }));
    } catch (error) {
        yield put(actions.fetchNetworkGraph.failure(error));
    }
}

function* filterEnvironmentPageBySearch() {
    const options = yield select(selectors.getEnvironmentSearchOptions);
    yield fork(getNetworkGraph, { options });
}

function* watchEnvironmentSearchOptions() {
    yield takeLatest(types.SET_SEARCH_OPTIONS, filterEnvironmentPageBySearch);
}

export default function* environment() {
    yield all([
        takeEveryNewlyMatchedLocation(environmentPath, filterEnvironmentPageBySearch),
        fork(watchEnvironmentSearchOptions)
    ]);
}
