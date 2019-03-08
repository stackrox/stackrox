import { all, take, takeLatest, call, fork, put, select, cancel } from 'redux-saga/effects';
import { delay } from 'redux-saga';
import { networkPath } from 'routePaths';
import * as service from 'services/NetworkService';
import { fetchClusters } from 'services/ClustersService';
import {
    actions as backendNetworkActions,
    types as backendNetworkTypes
} from 'reducers/network/backend';
import { types as dialogueNetworkTypes } from 'reducers/network/dialogue';
import { actions as graphNetworkActions, types as graphNetworkTypes } from 'reducers/network/graph';
import { types as pageNetworkTypes } from 'reducers/network/page';
import { types as searchNetworkTypes } from 'reducers/network/search';
import { actions as clusterActions } from 'reducers/clusters';
import { actions as notificationActions } from 'reducers/notifications';
import { selectors } from 'reducers';
import { takeEveryLocation } from 'utils/sagaEffects';
import { types as deploymentTypes } from 'reducers/deployments';
import { types as locationActionTypes } from 'reducers/routes';
import searchOptionsToQuery from 'services/searchOptionsToQuery';
import timeWindowToDate from 'utils/timeWindows';
import { getDeployment } from './deploymentSagas';

function* getNetworkFlowGraph(clusterId, query) {
    try {
        const timeWindow = yield select(selectors.getNetworkActivityTimeWindow);
        const flowResult = yield call(
            service.fetchNetworkFlowGraph,
            clusterId,
            query,
            timeWindowToDate(timeWindow)
        );
        yield put(backendNetworkActions.fetchNetworkFlowGraph.success(flowResult.response));
        yield put(graphNetworkActions.setNetworkFlowMapping(flowResult.response));
        yield put(graphNetworkActions.updateNetworkGraphTimestamp(new Date()));
    } catch (error) {
        yield put(backendNetworkActions.fetchNetworkFlowGraph.failure(error));
    }
}

function* getNetworkGraphs(clusterId, query, modification) {
    try {
        const policyResult = yield call(
            service.fetchNetworkPolicyGraph,
            clusterId,
            query,
            modification
        );
        yield put(backendNetworkActions.fetchNetworkPolicyGraph.success(policyResult.response));
        yield put(graphNetworkActions.updateNetworkGraphTimestamp(new Date()));
        yield fork(getNetworkFlowGraph, clusterId, query);
    } catch (error) {
        yield put(backendNetworkActions.fetchNetworkPolicyGraph.failure(error));
    }
}

function* getSelectedDeployment({ params }) {
    yield call(getDeployment, params);
}

export function* getNetworkPolicies({ params }) {
    try {
        const result = yield call(service.fetchNetworkPolicies, params);
        yield put(backendNetworkActions.fetchNetworkPolicies.success(result.response, { params }));
    } catch (error) {
        yield put(backendNetworkActions.fetchNetworkPolicies.failure(error));
    }
}

export function* pollNodeUpdates() {
    while (true) {
        try {
            const result = yield call(service.fetchNodeUpdates);
            yield put(backendNetworkActions.fetchNodeUpdates.success(result.response));
        } catch (error) {
            yield put(backendNetworkActions.fetchNodeUpdates.failure(error));
        }
        yield call(delay, 5000); // poll every 5 sec
    }
}

function* sendNetworkModificationNotification() {
    try {
        const clusterId = yield select(selectors.getSelectedNetworkClusterId);
        const notifierIds = yield select(selectors.getNetworkNotifiers);
        const modification = yield select(selectors.getNetworkPolicyModification);
        yield call(service.notifyNetworkPolicyModification, clusterId, notifierIds, modification);
        yield put(notificationActions.addNotification('Successfully sent notification.'));
        yield put(notificationActions.removeOldestNotification());
    } catch (error) {
        yield put(notificationActions.addNotification(error.response.data.error));
        yield put(notificationActions.removeOldestNotification());
    }
}

function* watchLocation() {
    let pollTask = null;
    while (true) {
        const action = yield take(locationActionTypes.LOCATION_CHANGE);
        const { payload: location } = action;

        if (
            location &&
            location.pathname &&
            location.pathname.startsWith(networkPath) &&
            !pollTask
        ) {
            // start only if it's not already in progress
            pollTask = yield fork(pollNodeUpdates);
        } else if (pollTask) {
            yield cancel(pollTask);
            pollTask = null;
            yield put(graphNetworkActions.setSelectedNodeId(null));
        }
    }
}

function* filterNetworkPageBySearch() {
    const clusterId = yield select(selectors.getSelectedNetworkClusterId);
    const searchOptions = yield select(selectors.getNetworkSearchOptions);
    const modification = yield select(selectors.getNetworkPolicyModification);
    if (searchOptions.length && searchOptions[searchOptions.length - 1].type) {
        return;
    }
    if (clusterId) {
        yield fork(getNetworkGraphs, clusterId, searchOptionsToQuery(searchOptions), modification);
    }
}

function* loadNetworkPage() {
    try {
        const result = yield call(fetchClusters);
        yield put(clusterActions.fetchClusters.success(result.response));
        yield put(graphNetworkActions.selectDefaultNetworkClusterId(result.response));
        yield fork(filterNetworkPageBySearch);
    } catch (error) {
        yield put(clusterActions.fetchClusters.failure(error));
    }
}

function* watchNetworkSearchOptions() {
    yield takeLatest(searchNetworkTypes.SET_SEARCH_OPTIONS, filterNetworkPageBySearch);
}

function* watchFetchDeploymentRequest() {
    yield takeLatest(deploymentTypes.FETCH_DEPLOYMENT.REQUEST, getSelectedDeployment);
}

function* watchNetworkPoliciesRequest() {
    yield takeLatest(backendNetworkTypes.FETCH_NETWORK_POLICIES.REQUEST, getNetworkPolicies);
}

function* watchSelectNetworkCluster() {
    yield takeLatest(graphNetworkTypes.SELECT_NETWORK_CLUSTER_ID, filterNetworkPageBySearch);
}

function* watchSetActivityTimeWindow() {
    yield takeLatest(pageNetworkTypes.SET_NETWORK_ACTIVITY_TIME_WINDOW, filterNetworkPageBySearch);
}

function* watchNetworkPolicyModification() {
    yield takeLatest(
        backendNetworkTypes.FETCH_NETWORK_POLICY_MODIFICATION.SUCCESS,
        filterNetworkPageBySearch
    );
}

function* watchNotifyNetworkPolicyModification() {
    yield takeLatest(
        dialogueNetworkTypes.SEND_POLICY_MODIFICATION_NOTIFICATION,
        sendNetworkModificationNotification
    );
}

function* watchNetworkNodesUpdate() {
    yield takeLatest(graphNetworkTypes.NETWORK_NODES_UPDATE, filterNetworkPageBySearch);
}

export default function* network() {
    yield all([
        takeEveryLocation(networkPath, loadNetworkPage),
        fork(watchNetworkSearchOptions),
        fork(watchNetworkPoliciesRequest),
        fork(watchFetchDeploymentRequest),
        fork(watchSelectNetworkCluster),
        fork(watchSetActivityTimeWindow),
        fork(watchNetworkNodesUpdate),
        fork(watchNetworkPolicyModification),
        fork(watchNotifyNetworkPolicyModification),
        fork(watchLocation)
    ]);
}
