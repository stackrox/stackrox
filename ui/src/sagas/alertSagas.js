import { all, takeLatest, call, fork, put, select } from 'redux-saga/effects';

import { dashboardPath } from 'routePaths';
import { takeEveryLocation } from 'utils/sagaEffects';
import * as service from 'services/AlertsService';
import { actions } from 'reducers/alerts';
import { types as dashboardTypes } from 'reducers/dashboard';
import { selectors } from 'reducers';
import searchOptionsToQuery from 'services/searchOptionsToQuery';

function* getGlobalAlertCounts(filters) {
    try {
        const newFilters = { ...filters };
        newFilters.group_by = 'CLUSTER';
        const result = yield call(service.fetchSummaryAlertCounts, newFilters);
        yield put(actions.fetchGlobalAlertCounts.success(result.response));
    } catch (error) {
        yield put(actions.fetchGlobalAlertCounts.failure(error));
    }
}

function* getAlertCountsByPolicyCategories(filters) {
    try {
        const newFilters = { ...filters };
        newFilters.group_by = 'CATEGORY';
        const result = yield call(service.fetchSummaryAlertCounts, newFilters);
        yield put(actions.fetchAlertCountsByPolicyCategories.success(result.response));
    } catch (error) {
        yield put(actions.fetchAlertCountsByPolicyCategories.failure(error));
    }
}

function* getAlertCountsByCluster(filters) {
    try {
        const newFilters = { ...filters };
        newFilters.group_by = 'CLUSTER';
        const result = yield call(service.fetchSummaryAlertCounts, newFilters);
        yield put(actions.fetchAlertCountsByCluster.success(result.response));
    } catch (error) {
        yield put(actions.fetchAlertCountsByCluster.failure(error));
    }
}

function* getAlertsByTimeseries(filters) {
    try {
        const result = yield call(service.fetchAlertsByTimeseries, filters);
        yield put(actions.fetchAlertsByTimeseries.success(result.response));
    } catch (error) {
        yield put(actions.fetchAlertsByTimeseries.failure(error));
    }
}

function* filterDashboardPageBySearch() {
    const searchOptions = yield select(selectors.getDashboardSearchOptions);
    if (searchOptions.length && searchOptions[searchOptions.length - 1].type) {
        return;
    }
    const filters = {
        query: searchOptionsToQuery(searchOptions)
    };
    const nestedFilter = {
        'request.query': searchOptionsToQuery(searchOptions)
    };
    yield fork(getGlobalAlertCounts, nestedFilter);
    yield fork(getAlertCountsByCluster, nestedFilter);
    yield fork(getAlertsByTimeseries, filters);
    yield fork(getAlertCountsByPolicyCategories, nestedFilter);
}

function* loadDashboardPage() {
    yield fork(filterDashboardPageBySearch);
}

function* watchDashboardSearchOptions() {
    yield takeLatest(dashboardTypes.SET_SEARCH_OPTIONS, filterDashboardPageBySearch);
}

export default function* alertsSaga() {
    yield all([
        takeEveryLocation(dashboardPath, loadDashboardPage),
        fork(watchDashboardSearchOptions)
    ]);
}
