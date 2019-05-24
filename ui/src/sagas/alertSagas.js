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
import { whitelistDeployments } from 'services/PoliciesService';
import { actions as notificationActions } from 'reducers/notifications';

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

function* whitelistDeploymentsFromAlerts(alertIds) {
    try {
        yield fork(cancelPolling);
        const alerts = yield all(alertIds.map(alertId => select(selectors.getAlert, alertId)));
        // need to produce { p_id1: [dep_id1], p_id2: [dep_id2, dep_id3] } to group deployments by policy
        const deploymentsGroupedByPolicy = alerts.reduce((acc, alert) => {
            const policyId = alert.policy.id;
            if (!acc[policyId]) acc[policyId] = [];
            acc[policyId].push(alert.deployment.name);
            return acc;
        }, {});
        return yield all(
            Object.keys(deploymentsGroupedByPolicy).map(policyId =>
                call(whitelistDeployments, policyId, deploymentsGroupedByPolicy[policyId])
            )
        );
    } finally {
        yield put(actions.pollAlerts.start());
    }
}

function* sendWhitelistDeployment({ params: alertId }) {
    try {
        const [result] = yield whitelistDeploymentsFromAlerts([alertId]);
        yield put(actions.whitelistDeployment.success(result.response));
    } catch (error) {
        yield put(actions.whitelistDeployment.failure(error));
    }
}

function* sendWhitelistDeployments({ params: alertIds }) {
    try {
        const results = yield whitelistDeploymentsFromAlerts(alertIds);
        yield put(actions.whitelistDeployments.success(results.map(r => r.response)));
    } catch (error) {
        yield put(actions.whitelistDeployments.failure(error));
    }
}

function* resolveAlerts({ alertIds, whitelist }) {
    try {
        yield fork(cancelPolling);
        yield call(service.resolveAlerts, alertIds, whitelist);
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
    yield takeLatest(types.WHITELIST_DEPLOYMENTS.REQUEST, sendWhitelistDeployments);
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

export default function* alertsSaga() {
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
