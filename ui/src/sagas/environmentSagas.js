import { all, takeLatest, call, fork, put, select } from 'redux-saga/effects';

import { environmentPath } from 'routePaths';
import * as service from 'services/EnvironmentService';
import { actions, types } from 'reducers/environment';
import { selectors } from 'reducers';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';
import searchOptionsToQuery from 'services/searchOptionsToQuery';

function* getNetworkGraph(filters) {
    try {
        const result = yield call(service.fetchEnvironmentGraph, filters);
        yield put(actions.fetchEnvironmentGraph.success(result.response));
    } catch (error) {
        yield put(actions.fetchEnvironmentGraph.failure(error));
    }
}

function* filterEnvironmentPageBySearch() {
    const searchOptions = yield select(selectors.getEnvironmentSearchOptions);
    const filters = {
        query: searchOptionsToQuery(searchOptions)
    };
    yield fork(getNetworkGraph, filters);
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
