import { all, take, takeLatest, call, fork, put, select, cancel } from 'redux-saga/effects';
import { delay } from 'redux-saga';
import { networkPath } from 'routePaths';
import * as service from 'services/NetworkService';
import { fetchClusters } from 'services/ClustersService';
import {
    actions as backendNetworkActions,
    types as backendNetworkTypes,
} from 'reducers/network/backend';
import { types as dialogueNetworkTypes } from 'reducers/network/dialogue';
import { actions as graphNetworkActions, types as graphNetworkTypes } from 'reducers/network/graph';
import { types as pageNetworkTypes } from 'reducers/network/page';
import { types as searchNetworkTypes } from 'reducers/network/search';
import {
    actions as wizardNetworkActions,
    types as wizardNetworkTypes,
} from 'reducers/network/sidepanel';
import {
    types as baselineSimulationTypes,
    actions as baselineSimulationActions,
} from 'reducers/network/baselineSimulation';
import { actions as clusterActions } from 'reducers/clusters';
import { actions as notificationActions } from 'reducers/notifications';
import { selectors } from 'reducers';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';
import { types as locationActionTypes } from 'reducers/routes';
import searchOptionsToQuery from 'services/searchOptionsToQuery';
import timeWindowToDate from 'utils/timeWindows';
import queryService from 'utils/queryService';
import { filterModes } from 'constants/networkFilterModes';

function* getFlowGraph(clusterId, namespaces, query, timeWindow, includePorts) {
    let flowGraph;
    try {
        const req = yield call(
            service.fetchNetworkFlowGraph,
            clusterId,
            namespaces,
            query,
            timeWindowToDate(timeWindow),
            includePorts
        );
        flowGraph = req.response;
    } catch (error) {
        yield put(backendNetworkActions.fetchNetworkFlowGraph.failure(error));
    }

    return flowGraph;
}

function* getPolicyGraph(clusterId, namespaces, query, modification, includePorts) {
    let policyGraph;
    try {
        const req = yield call(
            service.fetchNetworkPolicyGraph,
            clusterId,
            namespaces,
            query,
            modification,
            includePorts
        );
        policyGraph = req.response;
    } catch (error) {
        // On error, such as when an applied yaml is invalid, attempt to revert to the
        // previous successful policyGraph response
        yield put(notificationActions.addNotification(error.response.data.error));
        yield put(notificationActions.removeOldestNotification());
        policyGraph = yield select(selectors.getNetworkPolicyGraph);
    }

    return policyGraph;
}

function* getNetworkGraphs(clusterId, namespaces, query) {
    // Do not query the backend for network graph updates if no namespace is selected
    if (!namespaces.length) {
        return;
    }

    const timeWindow = yield select(selectors.getNetworkActivityTimeWindow);
    const modification = yield select(selectors.getNetworkPolicyModification);

    const flowGraph = yield call(getFlowGraph, clusterId, namespaces, query, timeWindow, true);
    const policyGraph = yield call(
        getPolicyGraph,
        clusterId,
        namespaces,
        query,
        modification,
        true
    );

    if (policyGraph && flowGraph) {
        yield put(graphNetworkActions.setNetworkEdgeMap(flowGraph, policyGraph));
        yield put(graphNetworkActions.setNetworkNodeMap(flowGraph, policyGraph));
        yield put(graphNetworkActions.updateNetworkGraphTimestamp(new Date()));
        yield put(backendNetworkActions.fetchNetworkFlowGraph.success(flowGraph));
        yield put(backendNetworkActions.fetchNetworkPolicyGraph.success(policyGraph));
    } else if (policyGraph) {
        yield put(backendNetworkActions.fetchNetworkPolicyGraph.success(policyGraph));
    }
}

export function* getActiveNetworkModification() {
    yield put(wizardNetworkActions.setNetworkPolicyModificationName('Active'));
    yield put(wizardNetworkActions.setNetworkPolicyModificationState('REQUEST'));
    try {
        const clusterId = yield select(selectors.getSelectedNetworkClusterId);
        const searchOptions = yield select(selectors.getNetworkSearchOptions);
        const modification = yield call(
            service.getActiveNetworkModification,
            clusterId,
            searchOptionsToQuery(searchOptions)
        );
        yield put(wizardNetworkActions.setNetworkPolicyModificationSource('ACTIVE'));
        yield put(wizardNetworkActions.setNetworkPolicyModification(modification));
        yield put(wizardNetworkActions.setNetworkPolicyModificationState('SUCCESS'));
    } catch (error) {
        yield put(wizardNetworkActions.setNetworkPolicyModificationState('ERROR'));
        yield put(notificationActions.addNotification(error.response.data.error));
        yield put(notificationActions.removeOldestNotification());
    }
}

export function* getUndoNetworkModification() {
    yield put(wizardNetworkActions.setNetworkPolicyModificationName('Undo'));
    yield put(wizardNetworkActions.setNetworkPolicyModificationState('REQUEST'));
    try {
        const clusterId = yield select(selectors.getSelectedNetworkClusterId);
        const modification = yield call(service.getUndoNetworkModification, clusterId);
        yield put(wizardNetworkActions.setNetworkPolicyModificationSource('UNDO'));
        yield put(wizardNetworkActions.setNetworkPolicyModification(modification));
        yield put(wizardNetworkActions.setNetworkPolicyModificationState('SUCCESS'));
    } catch (error) {
        yield put(wizardNetworkActions.setNetworkPolicyModificationState('ERROR'));
        yield put(notificationActions.addNotification(error.response.data.error));
        yield put(notificationActions.removeOldestNotification());
    }
}

// poll generators
export function* pollNodeUpdates() {
    while (true) {
        try {
            const clusterId = yield select(selectors.getSelectedNetworkClusterId);
            const result = yield call(service.fetchNodeUpdates, clusterId);
            yield put(backendNetworkActions.fetchNodeUpdates.success(result.response));
        } catch (error) {
            yield put(backendNetworkActions.fetchNodeUpdates.failure(error));
        }
        yield call(delay, 30000); // poll every 30 sec
    }
}

// send generators
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

function* sendNetworkModificationApplication() {
    try {
        const clusterId = yield select(selectors.getSelectedNetworkClusterId);
        const modification = yield select(selectors.getNetworkPolicyModification);
        yield call(service.applyNetworkPolicyModification, clusterId, modification);
        yield put(backendNetworkActions.applyNetworkPolicyModification.success());
        yield put(notificationActions.addNotification('Successfully applied YAML.'));
        yield put(notificationActions.removeOldestNotification());
        yield put(wizardNetworkActions.loadActiveNetworkPolicyModification());
    } catch (error) {
        yield put(backendNetworkActions.applyNetworkPolicyModification.failure(error));
        yield put(notificationActions.addNotification(error.response.data.error));
        yield put(notificationActions.removeOldestNotification());
    }
}

// misc action generators
function* filterNetworkPageBySearch() {
    const clusterId = yield select(selectors.getSelectedNetworkClusterId);
    const selectedNamespaces = yield select(selectors.getSelectedNamespaceFilters);
    const searchOptions = yield select(selectors.getNetworkSearchOptions);
    if (searchOptions.length && searchOptions[searchOptions.length - 1].type) {
        return;
    }
    if (clusterId) {
        const filters = searchOptionsToQuery(searchOptions);
        yield fork(getNetworkGraphs, clusterId, selectedNamespaces, filters);
    }
}

function* filterNetworkPageByBaselineSimulation() {
    const clusterId = yield select(selectors.getSelectedNetworkClusterId);
    const selectedNamespaces = yield select(selectors.getSelectedNamespaceFilters);
    const node = yield select(selectors.getSelectedNode);
    const filter = queryService.objectToWhereClause({ Deployment: node.name });
    yield fork(getNetworkGraphs, clusterId, selectedNamespaces, filter);
    /*
     * We want to switch to the allowed filter when showing the simulated state because we want to
     * show the possible added, removed, modified edges
     */
    yield put(graphNetworkActions.setNetworkGraphFilterMode(filterModes.allowed));
}

function* getBaselineComparisons() {
    const { deploymentId } = yield select(selectors.getSelectedNode);
    try {
        // TODO: We could probably do this in parallel to make it more efficient
        // Saif found this:
        // https://redux-saga.js.org/docs/advanced/RunningTasksInParallel/
        yield put(baselineSimulationActions.fetchBaselineComparisons.request());
        const baselineResponse = yield call(service.fetchBaselineComparison, { deploymentId });
        yield put(baselineSimulationActions.fetchBaselineComparisons.success(baselineResponse));
    } catch (error) {
        yield put(baselineSimulationActions.fetchBaselineComparisons.failure(error));
    }
    try {
        yield put(baselineSimulationActions.fetchUndoComparisons.request());
        const undoResponse = yield call(service.fetchUndoComparison, { deploymentId });
        yield put(baselineSimulationActions.fetchUndoComparisons.success(undoResponse));
    } catch (error) {
        yield put(baselineSimulationActions.fetchUndoComparisons.failure(error));
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

function* generateNetworkModification() {
    yield put(wizardNetworkActions.setNetworkPolicyModificationName('StackRox Generated'));
    yield put(wizardNetworkActions.setNetworkPolicyModificationState('REQUEST'));
    try {
        const clusterId = yield select(selectors.getSelectedNetworkClusterId);
        const searchOptions = yield select(selectors.getNetworkSearchOptions);
        const timeWindow = yield select(selectors.getNetworkActivityTimeWindow);
        const excludePortsProtocols = yield select(
            selectors.getNetworkPolicyExcludePortsProtocolsState
        );
        const modification = yield call(
            service.generateNetworkModification,
            clusterId,
            searchOptionsToQuery(searchOptions),
            timeWindowToDate(timeWindow),
            excludePortsProtocols
        );
        yield put(wizardNetworkActions.setNetworkPolicyModificationSource('GENERATED'));
        yield put(wizardNetworkActions.setNetworkPolicyModification(modification));
        yield put(wizardNetworkActions.setNetworkPolicyModificationState('SUCCESS'));
        yield put(notificationActions.addNotification('Successfully generated YAML.'));
        yield put(notificationActions.removeOldestNotification());
    } catch (error) {
        yield put(wizardNetworkActions.setNetworkPolicyModificationState('ERROR'));
        yield put(notificationActions.addNotification(error.response.data.error));
        yield put(notificationActions.removeOldestNotification());
    }
}

function* resetNamespaceFilters() {
    yield put(graphNetworkActions.setSelectedNamespaceFilters([]));
}

// watch generators
function* watchLocation() {
    let pollTask = null;
    while (true) {
        const action = yield take(locationActionTypes.LOCATION_CHANGE);
        const {
            payload: { location },
        } = action;
        const onNetworkPage = location.pathname?.startsWith('/main/network');

        if (onNetworkPage && !pollTask) {
            // start only if it's not already in progress
            pollTask = yield fork(pollNodeUpdates);
        } else if (!onNetworkPage && pollTask) {
            // cancel when navigating away from network page
            yield cancel(pollTask);
            pollTask = null;
            yield put(graphNetworkActions.setSelectedNode(null));
        }
    }
}

function* watchNetworkSearchOptions() {
    yield takeLatest(searchNetworkTypes.SET_SEARCH_OPTIONS, filterNetworkPageBySearch);
}

function* watchApplyNetworkPolicyModification() {
    yield takeLatest(
        backendNetworkTypes.APPLY_NETWORK_POLICY_MODIFICATION.REQUEST,
        sendNetworkModificationApplication
    );
}

function* watchActiveNetworkModification() {
    yield takeLatest(
        wizardNetworkTypes.LOAD_ACTIVE_NETWORK_POLICY_MODIFICATION,
        getActiveNetworkModification
    );
}

function* watchUndoNetworkModification() {
    yield takeLatest(
        wizardNetworkTypes.LOAD_UNDO_NETWORK_POLICY_MODIFICATION,
        getUndoNetworkModification
    );
}

function* watchGenerateNetworkModification() {
    yield takeLatest(
        wizardNetworkTypes.GENERATE_NETWORK_POLICY_MODIFICATION,
        generateNetworkModification
    );
}

function* watchSelectNetworkCluster() {
    yield all([
        takeLatest(graphNetworkTypes.SELECT_NETWORK_CLUSTER_ID, filterNetworkPageBySearch),
        takeLatest(graphNetworkTypes.SELECT_NETWORK_CLUSTER_ID, resetNamespaceFilters),
    ]);
}

function* watchSetSelectNamespaceFilters() {
    yield takeLatest(graphNetworkTypes.SET_SELECTED_NAMESPACE_FILTERS, filterNetworkPageBySearch);
}

function* watchSetActivityTimeWindow() {
    yield takeLatest(pageNetworkTypes.SET_NETWORK_ACTIVITY_TIME_WINDOW, filterNetworkPageBySearch);
}

function* watchNetworkPolicyModification() {
    yield takeLatest(wizardNetworkTypes.SET_POLICY_MODIFICATION, filterNetworkPageBySearch);
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

function* watchBaselineSimulation() {
    yield takeLatest(baselineSimulationTypes.START_BASELINE_SIMULATION, getBaselineComparisons);
    yield takeLatest(
        baselineSimulationTypes.START_BASELINE_SIMULATION,
        filterNetworkPageByBaselineSimulation
    );
    yield takeLatest(baselineSimulationTypes.STOP_BASELINE_SIMULATION, filterNetworkPageBySearch);
}

// all generators
export default function* network() {
    yield all([
        takeEveryNewlyMatchedLocation(networkPath, loadNetworkPage),
        fork(watchNetworkSearchOptions),
        fork(watchActiveNetworkModification),
        fork(watchUndoNetworkModification),
        fork(watchGenerateNetworkModification),
        fork(watchSelectNetworkCluster),
        fork(watchSetSelectNamespaceFilters),
        fork(watchSetActivityTimeWindow),
        fork(watchNetworkNodesUpdate),
        fork(watchNetworkPolicyModification),
        fork(watchNotifyNetworkPolicyModification),
        fork(watchApplyNetworkPolicyModification),
        fork(watchBaselineSimulation),
        fork(watchLocation),
    ]);
}
