import { all, takeLatest, call, fork, put, select } from 'redux-saga/effects';

import watchLocation from 'utils/watchLocation';
import { fetchDeployments, fetchDeployment } from 'services/DeploymentsService';
import { actions, types } from 'reducers/deployments';
import { types as dashboardTypes } from 'reducers/dashboard';
import { selectors } from 'reducers';

const riskPath = '/main/risk/:deploymentId?';
const dashboardPath = '/main/dashboard';
const policiesPath = '/main/policies';

export function* getDeployments({ options = [] }) {
    try {
        const result = yield call(fetchDeployments, options);
        yield put(actions.fetchDeployments.success(result.response, { options }));
    } catch (error) {
        yield put(actions.fetchDeployments.failure(error));
    }
}

export function* getDeployment(id) {
    yield put(actions.fetchDeployment.request());
    try {
        const result = yield call(fetchDeployment, id);
        yield put(actions.fetchDeployment.success(result.response, { id }));
    } catch (error) {
        yield put(actions.fetchDeployment.failure(error));
    }
}

function* filterDashboardPageBySearch() {
    const options = yield select(selectors.getDashboardSearchOptions);
    yield fork(getDeployments, { options });
}

function* filterPoliciesPageBySearch() {
    const options = yield select(selectors.getPoliciesSearchOptions);
    yield fork(getDeployments, { options });
}

function* filterRiskPageBySearch() {
    const options = yield select(selectors.getDeploymentsSearchOptions);
    yield fork(getDeployments, { options });
}

function* loadRiskPage(match) {
    yield fork(filterRiskPageBySearch);

    const { deploymentId } = match.params;
    if (deploymentId) {
        yield fork(getDeployment, deploymentId);
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
        fork(watchLocation, dashboardPath, filterDashboardPageBySearch),
        fork(watchLocation, policiesPath, filterPoliciesPageBySearch),
        fork(watchLocation, riskPath, loadRiskPage),
        fork(watchDeploymentsSearchOptions),
        fork(watchDashboardSearchOptions)
    ]);
}
