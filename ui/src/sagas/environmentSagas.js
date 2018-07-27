import { all, take, takeLatest, call, fork, put, select, cancel } from 'redux-saga/effects';
import { delay } from 'redux-saga';

import { selectors } from 'reducers';

import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';
import searchOptionsToQuery from 'services/searchOptionsToQuery';
import { environmentPath } from 'routePaths';
import * as service from 'services/EnvironmentService';
import { actions, types } from 'reducers/environment';
import { types as deploymentTypes } from 'reducers/deployments';
import { types as locationActionTypes } from 'reducers/routes';
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

export function* pollNodeUpdates() {
    while (true) {
        try {
            const result = yield call(service.fetchNodeUpdates);
            yield put(actions.fetchNodeUpdates.success(result.response));
        } catch (error) {
            yield put(actions.fetchNodeUpdates.failure(error));
        }
        yield call(delay, 5000); // poll every 5 sec
    }
}

function* watchLocation() {
    let pollTask = null;
    while (true) {
        const action = yield take(locationActionTypes.LOCATION_CHANGE);
        const { payload: location } = action;

        if (location && location.pathname && location.pathname.startsWith(environmentPath)) {
            // start only if it's not already in progress
            if (!pollTask) {
                pollTask = yield fork(pollNodeUpdates);
            }
        } else if (pollTask) {
            yield cancel(pollTask);
            pollTask = null;
        }
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
        fork(watchFetchDeploymentRequest),
        fork(watchLocation)
    ]);
}
