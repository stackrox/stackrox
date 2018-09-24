import { all, take, takeLatest, call, fork, put, select, cancel } from 'redux-saga/effects';
import { delay } from 'redux-saga';
import { environmentPath, networkPath } from 'routePaths';
import * as service from 'services/EnvironmentService';
import { fetchClusters } from 'services/ClustersService';
import { actions, types } from 'reducers/environment';
import { actions as clusterActions, types as clusterTypes } from 'reducers/clusters';
import { actions as notificationActions } from 'reducers/notifications';
import { selectors } from 'reducers';
import { takeEveryLocation } from 'utils/sagaEffects';
import searchOptionsToQuery from 'services/searchOptionsToQuery';
import { types as deploymentTypes } from 'reducers/deployments';
import { types as locationActionTypes } from 'reducers/routes';
import { getDeployment } from './deploymentSagas';

function* getNetworkGraph(filters, clusterId) {
    try {
        const result = yield call(service.fetchEnvironmentGraph, filters, clusterId);
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

function* sendYAMLNotification({ notifierId }) {
    try {
        const clusterId = yield select(selectors.getSelectedEnvironmentClusterId);
        const { content } = yield select(selectors.getYamlFile);
        yield call(service.sendYAMLNotification, clusterId, notifierId, content);
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
            location.pathname.startsWith(environmentPath) &&
            !pollTask
        ) {
            // start only if it's not already in progress
            pollTask = yield fork(pollNodeUpdates);
        } else if (pollTask) {
            yield cancel(pollTask);
            pollTask = null;
            yield put(actions.setSelectedNodeId(null));
            yield put(actions.setSimulatorMode(false));
            yield put(actions.setNetworkGraphState(null));
            yield put(actions.setYamlFile(null));
        }
    }
}

function* getClusters() {
    try {
        const result = yield call(fetchClusters);
        yield put(clusterActions.fetchClusters.success(result.response));
    } catch (error) {
        yield put(clusterActions.fetchClusters.failure(error));
    }
}

function* filterEnvironmentPageBySearch() {
    const clusterId = yield select(selectors.getSelectedEnvironmentClusterId);
    const searchOptions = yield select(selectors.getEnvironmentSearchOptions);
    const yamlFile = yield select(selectors.getYamlFile);
    const simulatorMode = yield select(selectors.getSimulatorMode);
    if (searchOptions.length && searchOptions[searchOptions.length - 1].type) {
        return;
    }
    const filters = {
        query: searchOptionsToQuery(searchOptions)
    };
    if (simulatorMode && yamlFile) {
        filters.simulationYaml = yamlFile.content;
    }
    if (clusterId) {
        yield fork(getNetworkGraph, filters, clusterId);
    }
}

function* loadEnvironmentPage() {
    yield fork(getClusters);
    yield fork(filterEnvironmentPageBySearch);
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

function* watchSelectEnvironmentCluster() {
    yield takeLatest(types.SELECT_ENVIRONMENT_CLUSTER_ID, filterEnvironmentPageBySearch);
}

function* watchSendYAMLNotification() {
    yield takeLatest(types.SEND_YAML_NOTIFICATION, sendYAMLNotification);
}

function* watchFetchClustersSuccess() {
    yield takeLatest(clusterTypes.FETCH_CLUSTERS.SUCCESS, filterEnvironmentPageBySearch);
}

function* watchSetYamlFile() {
    yield takeLatest(types.SET_YAML_FILE, filterEnvironmentPageBySearch);
}

export default function* environment() {
    yield all([
        takeEveryLocation(environmentPath, loadEnvironmentPage),
        takeEveryLocation(networkPath, loadEnvironmentPage),
        fork(watchEnvironmentSearchOptions),
        fork(watchFetchEnvironmentGraphRequest),
        fork(watchNetworkPoliciesRequest),
        fork(watchFetchDeploymentRequest),
        fork(watchSelectEnvironmentCluster),
        fork(watchFetchClustersSuccess),
        fork(watchSetYamlFile),
        fork(watchSendYAMLNotification),
        fork(watchLocation)
    ]);
}
