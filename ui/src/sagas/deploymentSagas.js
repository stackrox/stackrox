import { all, takeLatest, call, fork, put, select } from 'redux-saga/effects';

import watchLocation from 'utils/watchLocation';
import { fetchDeployments, fetchDeployment } from 'services/DeploymentsService';
import { actions, types } from 'reducers/deployments';
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
    try {
        const result = yield call(fetchDeployment, id);
        yield put(actions.fetchDeployment.success(result.response, { id }));
    } catch (error) {
        yield put(actions.fetchDeployment.failure(error));
    }
}

function* watchDeploymentsSearchOptions() {
    yield takeLatest(types.SET_SEARCH_OPTIONS, getDeployments);
}

function* getAllDeployments() {
    yield call(getDeployments, {});
}

function* loadRiskPage(match) {
    const options = yield select(selectors.getDeploymentsSearchOptions);
    yield fork(getDeployments, { options });

    const { deploymentId } = match.params;
    if (deploymentId) {
        yield fork(getDeployment, deploymentId);
    }
}

export default function* deployments() {
    yield all([
        fork(watchLocation, dashboardPath, getAllDeployments),
        fork(watchLocation, policiesPath, getAllDeployments),
        fork(watchLocation, riskPath, loadRiskPage),
        fork(watchDeploymentsSearchOptions)
    ]);
}
