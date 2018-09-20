import { all, takeLatest, call, fork, put, select } from 'redux-saga/effects';

import { riskPath, dashboardPath, policiesPath } from 'routePaths';
import { fetchDeployments, fetchDeployment } from 'services/DeploymentsService';
import { actions, types } from 'reducers/deployments';
import { types as dashboardTypes } from 'reducers/dashboard';
import { selectors } from 'reducers';
import { takeEveryLocation, takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';

export function* getDeployments({ options = [] }) {
    try {
        const result = yield call(fetchDeployments, options);
        yield put(actions.fetchDeployments.success(result.response, { options }));
    } catch (error) {
        yield put(actions.fetchDeployments.failure(error));
    }
}

export function* getDeployment(id) {
    try {
        const result = yield call(fetchDeployment, id);
        yield put(actions.fetchDeployment.success(result.response, { id }));
    } catch (error) {
        yield put(actions.fetchDeployment.failure(error));
    }
}

function* filterDashboardPageBySearch() {
    const options = yield select(selectors.getDashboardSearchOptions);
    if (options.length && options[options.length - 1].type) {
        return;
    }
    yield fork(getDeployments, { options });
}

function* filterPoliciesPageBySearch() {
    const options = yield select(selectors.getPoliciesSearchOptions);
    if (options.length && options[options.length - 1].type) {
        return;
    }
    yield fork(getDeployments, { options });
}

function* filterRiskPageBySearch() {
    const options = yield select(selectors.getDeploymentsSearchOptions);
    if (options.length && options[options.length - 1].type) {
        return;
    }
    yield fork(getDeployments, { options });
}

function* getSelectedDeployment({ match }) {
    const { deploymentId } = match.params;
    if (deploymentId) {
        yield put(actions.fetchDeployment.request());
        yield call(getDeployment, deploymentId);
    }
}

function* watchDeploymentsSearchOptions() {
    yield takeLatest(types.SET_SEARCH_OPTIONS, filterRiskPageBySearch);
}

function* watchDashboardSearchOptions() {
    yield takeLatest(dashboardTypes.SET_SEARCH_OPTIONS, filterDashboardPageBySearch);
}

export default function* deployments() {
    yield all([
        takeEveryNewlyMatchedLocation(dashboardPath, filterDashboardPageBySearch),
        takeEveryNewlyMatchedLocation(policiesPath, filterPoliciesPageBySearch),
        takeEveryNewlyMatchedLocation(riskPath, filterRiskPageBySearch),
        takeEveryLocation(riskPath, getSelectedDeployment),
        fork(watchDeploymentsSearchOptions),
        fork(watchDashboardSearchOptions)
    ]);
}
