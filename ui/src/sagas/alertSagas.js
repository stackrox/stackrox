import { delay } from 'redux-saga';
import { all, take, takeLatest, call, cancel, fork, put, select } from 'redux-saga/effects';
import queryString from 'query-string';

import * as service from 'services/AlertsService';
import { actions, types } from 'reducers/alerts';
import { types as dashboardTypes } from 'reducers/dashboard';
import { types as locationActionTypes } from 'reducers/routes';
import { selectors } from 'reducers';

const dashboardPath = '/main/dashboard';
const violationsPath = '/main/violations';

function* getAlert({ params: alertId }) {
    try {
        const result = yield call(service.fetchAlert, alertId);
        yield put(actions.fetchAlert.success(result.response));
    } catch (error) {
        yield put(actions.fetchAlert.failure(error, alertId));
    }
}

function* getAlertNumsByPolicy() {
    try {
        const searchQuery = yield select(selectors.getAlertsSearchQuery);
        const newFilters = {};
        newFilters.query = searchQuery;
        const result = yield call(service.fetchAlertNumsByPolicy, newFilters);
        yield put(actions.fetchAlertNumsByPolicy.success(result.response));
    } catch (error) {
        yield put(actions.fetchAlertNumsByPolicy.failure(error));
    }
}

function* getGlobalAlertCounts(filters) {
    try {
        const newFilters = { ...filters };
        newFilters.group_by = 'CLUSTER';
        const result = yield call(service.fetchAlertCounts, newFilters);
        yield put(actions.fetchGlobalAlertCounts.success(result.response));
    } catch (error) {
        yield put(actions.fetchGlobalAlertCounts.failure(error));
    }
}

function* getAlertCountsByPolicyCategories(filters) {
    try {
        const newFilters = { ...filters };
        newFilters.group_by = 'CATEGORY';
        const result = yield call(service.fetchAlertCounts, newFilters);
        yield put(actions.fetchAlertCountsByPolicyCategories.success(result.response));
    } catch (error) {
        yield put(actions.fetchAlertCountsByPolicyCategories.failure(error));
    }
}

function* getAlertCountsByCluster(filters) {
    try {
        const newFilters = { ...filters };
        newFilters.group_by = 'CLUSTER';
        const result = yield call(service.fetchAlertCounts, newFilters);
        /*
         * @TODO This is a hack. Will need to remove it. Backend API should allow filtering the response using the search query
         */
        const filteredResult = Object.assign({}, result);
        if (filters && filters.query) {
            const clusterName = filters.query.replace('Cluster:', '');
            if (clusterName)
                filteredResult.response.groups = result.response.groups.filter(
                    obj => obj.group === clusterName
                );
        }
        yield put(actions.fetchAlertCountsByCluster.success(result.response));
    } catch (error) {
        yield put(actions.fetchAlertCountsByCluster.failure(error));
    }
}

function* getAlertsByTimeseries(filters) {
    try {
        const result = yield call(service.fetchAlertsByTimeseries, filters);
        /*
         * @TODO This is a hack. Will need to remove it. Backend API should allow filtering the response using the search query
         */
        const filteredResult = Object.assign({}, result);
        if (filters && filters.query) {
            const clusterName = filters.query.replace('Cluster:', '');
            if (clusterName)
                filteredResult.response.clusters = result.response.clusters.filter(
                    obj => obj.cluster === clusterName
                );
        }
        yield put(actions.fetchAlertsByTimeseries.success(result.response));
    } catch (error) {
        yield put(actions.fetchAlertsByTimeseries.failure(error));
    }
}

function* getAlertsByPolicy() {
    const policyId = yield select(selectors.getSelectedViolatedPolicyId);
    if (!policyId) return;
    try {
        const result = yield call(service.fetchAlertsByPolicy, policyId);
        yield put(actions.fetchAlertsByPolicy.success(result.response, policyId));
    } catch (error) {
        yield put(actions.fetchAlertsByPolicy.failure(error, policyId));
    }
}

function* pollAlertsByPolicy() {
    while (true) {
        let failsCount = 0;
        try {
            yield all([call(getAlertNumsByPolicy), call(getAlertsByPolicy)]);
            failsCount = 0;
        } catch (err) {
            console.error('Error during alerts polling', err);
            failsCount += 1;
            if (failsCount === 2) {
                // complain when retry didn't help
                yield put(actions.fetchAlertsByPolicy.failure('Cannot reach the server.'));
            }
        }
        yield delay(5000); // poll every 5 sec
    }
}

function* filterDashboardPageBySearch() {
    const searchQuery = yield select(selectors.getDashboardSearchQuery);
    const filters = {
        query: searchQuery
    };
    yield fork(getGlobalAlertCounts, {});
    yield fork(getAlertCountsByPolicyCategories, filters);
    yield fork(getAlertCountsByCluster, filters);
    yield fork(getAlertsByTimeseries, filters);
}

function* watchLocation() {
    let pollTask;
    while (true) {
        // it's a tricky/hack-y behavior here when deployment whitelisting happens: UI closes the dialog,
        // it causes location to update and therefore we're re-fetching everything for alerts
        const action = yield take(locationActionTypes.LOCATION_CHANGE);
        const { payload: location } = action;

        if (pollTask) yield cancel(pollTask); // cancel polling in any case

        if (location && location.pathname && location.pathname.startsWith(violationsPath)) {
            pollTask = yield fork(pollAlertsByPolicy, queryString.parse(location.search));
        } else if (location && location.pathname && location.pathname.startsWith(dashboardPath)) {
            yield fork(filterDashboardPageBySearch);
        }
    }
}

function* watchAlertRequest() {
    yield takeLatest(types.FETCH_ALERT.REQUEST, getAlert);
}

function* watchSelectedViolatedPolicy() {
    yield takeLatest(types.SELECT_VIOLATED_POLICY, getAlertsByPolicy);
}

function* watchAlertsSearchOptions() {
    const action = yield take(locationActionTypes.LOCATION_CHANGE);
    const { payload: location } = action;
    yield takeLatest(
        types.SET_SEARCH_OPTIONS,
        getAlertNumsByPolicy,
        queryString.parse(location.search)
    );
}

function* watchDashboardSearchOptions() {
    yield takeLatest(dashboardTypes.SET_SEARCH_OPTIONS, filterDashboardPageBySearch);
}

export default function* alerts() {
    yield all([
        fork(watchLocation),
        fork(watchSelectedViolatedPolicy),
        fork(watchAlertRequest),
        fork(watchAlertsSearchOptions),
        fork(watchDashboardSearchOptions)
    ]);
}
