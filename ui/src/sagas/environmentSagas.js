import { all, takeLatest, call, fork, put, select } from 'redux-saga/effects';

import { selectors } from 'reducers';

import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';
import searchOptionsToQuery from 'services/searchOptionsToQuery';
import { environmentPath } from 'routePaths';
import * as service from 'services/EnvironmentService';
import { actions, types } from 'reducers/environment';
import { types as deploymentTypes } from 'reducers/deployments';
import { getDeployment } from './deploymentSagas';

function* getNetworkGraph(filters) {
    try {
        const result = yield call(service.fetchEnvironmentGraph, filters);
        yield put(actions.fetchEnvironmentGraph.success(result.response));
    } catch (error) {
        yield put(actions.fetchEnvironmentGraph.failure(error));
    }
}

function* getSelectedDeployment({ params }) {
    yield call(getDeployment, params);
}

export function* getNetworkPolicies({ params }) {
    try {
        const result = yield call(service.fetchNetworkPolicies, params);
        yield put(actions.fetchNetworkPolicies.success(result.response, { params }));
    } catch (error) {
        yield put(actions.fetchNetworkPolicies.failure(error));
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

function* watchFetchEnvironmentGraphRequest() {
    yield takeLatest(types.FETCH_ENVIRONMENT_GRAPH.REQUEST, filterEnvironmentPageBySearch);
}

function* watchFetchDeploymentRequest() {
    yield takeLatest(deploymentTypes.FETCH_DEPLOYMENT.REQUEST, getSelectedDeployment);
}

function* watchNetworkPoliciesRequest() {
    yield takeLatest(types.FETCH_NETWORK_POLICIES.REQUEST, getNetworkPolicies);
}

export default function* environment() {
    yield all([
        takeEveryNewlyMatchedLocation(environmentPath, filterEnvironmentPageBySearch),
        fork(watchEnvironmentSearchOptions),
        fork(watchFetchEnvironmentGraphRequest),
        fork(watchNetworkPoliciesRequest),
        fork(watchFetchDeploymentRequest)
    ]);
}
