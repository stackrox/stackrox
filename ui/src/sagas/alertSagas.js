import {
    all,
    take,
    takeLatest,
    call,
    fork,
    put,
    select,
    race,
    takeEvery
} from 'redux-saga/effects';
import { delay } from 'redux-saga';
import { LOCATION_CHANGE } from 'react-router-redux';

import { violationsPath, dashboardPath } from 'routePaths';
import { takeEveryLocation } from 'utils/sagaEffects';
import * as service from 'services/AlertsService';
import { actions, types } from 'reducers/alerts';
import { types as dashboardTypes } from 'reducers/dashboard';
import { selectors } from 'reducers';
import searchOptionsToQuery from 'services/searchOptionsToQuery';
import { whitelistDeployment } from 'services/PoliciesService';
import { actions as notificationActions } from 'reducers/notifications';

function filterTimeseriesResultsByClusterSearchOptions(result, filters) {
    const filteredResult = Object.assign({}, result);
    if (filters && filters.query) {
        let clusterNames = filters.query.split('+').filter(obj => obj.includes('Cluster:'));
        if (clusterNames.length) clusterNames = clusterNames[0].replace('Cluster:', '').split(',');
        if (clusterNames.length && clusterNames[0] !== '') {
            filteredResult.response.clusters = result.response.clusters.filter(obj =>
                clusterNames.includes(obj.cluster)
            );
        }
    }
    return filteredResult;
}

function filterCountsResultByClusterSearchOption(result, filters) {
    const filteredResult = Object.assign({}, result);
    if (filters && filters.query) {
        let clusterNames = filters.query.split('+').filter(obj => obj.includes('Cluster:'));
        if (clusterNames.length) clusterNames = clusterNames.replace('Cluster:', '').split(',');
        if (clusterNames.length && clusterNames[0] !== '')
            filteredResult.response.groups = result.response.groups.filter(obj =>
                clusterNames.includes(obj.group)
            );
    }
    return filteredResult;
}

function* getAlerts(filters) {
    try {
        const result = yield call(service.fetchAlerts, filters);
        yield put(actions.fetchAlerts.success(result.response));
    } catch (error) {
        yield put(actions.fetchAlerts.failure(error));
    }
}

function* getAlert(id) {
    try {
        const result = yield call(service.fetchAlert, id);
        yield put(actions.fetchAlert.success(result.response, { id }));
    } catch (error) {
        yield put(actions.fetchAlert.failure(error));
    }
}

function* getGlobalAlertCounts(filters) {
    try {
        const newFilters = { ...filters };
        newFilters.group_by = 'CLUSTER';
        const result = yield call(service.fetchAlertCounts, newFilters);
        /*
         * @TODO This is a hack. Will need to remove it. Backend API should allow filtering the response using the search query
         */
        const filteredResult = filterCountsResultByClusterSearchOption(result, filters);
        yield put(actions.fetchGlobalAlertCounts.success(filteredResult.response));
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
        const filteredResult = filterCountsResultByClusterSearchOption(result, filters);
        yield put(actions.fetchAlertCountsByCluster.success(filteredResult.response));
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
        const filteredResult = filterTimeseriesResultsByClusterSearchOptions(result, filters);
        yield put(actions.fetchAlertsByTimeseries.success(filteredResult.response));
    } catch (error) {
        yield put(actions.fetchAlertsByTimeseries.failure(error));
    }
}

function* filterViolationsPageBySearch() {
    const searchOptions = yield select(selectors.getAlertsSearchOptions);
    if (searchOptions.length && searchOptions[searchOptions.length - 1].type) {
        return;
    }
    const filters = {
        query: searchOptionsToQuery(searchOptions)
    };
    yield fork(getAlerts, filters);
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

function* loadViolationsPage({ match }) {
    yield put(actions.pollAlerts.start());
    const { alertId } = match.params;
    if (alertId) {
        yield fork(getAlert, alertId);
    }
}

function* loadDashboardPage() {
    yield fork(filterDashboardPageBySearch);
}

function* pollAlerts() {
    while (true) {
        let failsCount = 0;
        try {
            yield all([call(filterViolationsPageBySearch)]);
            failsCount = 0;
        } catch (err) {
            failsCount += 1;
            if (failsCount === 2) {
                // complain when retry didn't help
                yield put(actions.fetchAlerts.failure('Cannot reach the server.'));
            }
        }
        yield delay(5000); // poll every 5 sec
    }
}

// place all actions to stop polling in this function
function* cancelPolling() {
    yield put(actions.pollAlerts.stop());
}

function* sendWhitelistDeployment({ params: alertId }) {
    try {
        yield fork(cancelPolling);
        const alert = yield select(selectors.getAlert, alertId);
        const result = yield call(whitelistDeployment, alert.policy.id, alert.deployment.name);
        yield put(actions.whitelistDeployment.success(result.response));
        yield fork(pollAlerts);
    } catch (error) {
        yield put(actions.whitelistDeployment.failure(error));
    }
}

function* resolveAlerts({ alertIds }) {
    try {
        yield fork(cancelPolling);
        yield call(service.resolveAlerts, alertIds);
        yield fork(pollAlerts);
    } catch (error) {
        yield put(notificationActions.addNotification(error.response.data.error));
        yield put(notificationActions.removeOldestNotification());
    }
}

function* watchAlertsSearchOptions() {
    yield takeLatest(types.SET_SEARCH_OPTIONS, filterViolationsPageBySearch);
}

function* watchDashboardSearchOptions() {
    yield takeLatest(dashboardTypes.SET_SEARCH_OPTIONS, filterDashboardPageBySearch);
}

function* watchWhitelistDeployment() {
    yield takeLatest(types.WHITELIST_DEPLOYMENT.REQUEST, sendWhitelistDeployment);
}

function* watchResolveAlerts() {
    yield takeLatest(types.RESOLVE_ALERTS, resolveAlerts);
}

function* pollSagaWatcher() {
    while (true) {
        yield take(types.POLL_ALERTS.START);
        yield race([call(pollAlerts), take(types.POLL_ALERTS.STOP)]);
    }
}

export default function* alerts() {
    yield all([
        takeEvery(LOCATION_CHANGE, cancelPolling),
        takeEveryLocation(violationsPath, loadViolationsPage),
        takeEveryLocation(dashboardPath, loadDashboardPage),
        fork(watchAlertsSearchOptions),
        fork(watchDashboardSearchOptions),
        fork(watchWhitelistDeployment),
        fork(watchResolveAlerts),
        fork(pollSagaWatcher)
    ]);
}
