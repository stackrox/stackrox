import { take, takeLatest, call, fork, put, all, select, race } from 'redux-saga/effects';
import { delay } from 'redux-saga';
import { getFormValues } from 'redux-form';
import Raven from 'raven-js';

import * as service from 'services/ClustersService';
import { actions, types, wizardPages, clusterFormId } from 'reducers/clusters';
import { actions as notificationActions } from 'reducers/notifications';
import { selectors } from 'reducers';

function* getClusters() {
    try {
        const result = yield call(service.fetchClusters);
        yield put(actions.fetchClusters.success(result.response));
    } catch (error) {
        yield put(actions.fetchClusters.failure(error));
    }
}

function* getCluster(clusterId) {
    try {
        const result = yield call(service.fetchCluster, clusterId);
        yield put(actions.fetchCluster.success(result.response));
    } catch (error) {
        yield put(actions.fetchCluster.failure(error));
    }
}

function* saveCluster(cluster) {
    try {
        const result = yield call(service.saveCluster, cluster);
        yield put(actions.saveCluster.success(result.response));
        return result.response.result.cluster; // that will be cluster ID
    } catch (error) {
        if (error.response) {
            yield put(notificationActions.addNotification(error.response.data.error));
            yield put(notificationActions.removeOldestNotification());
        }
        yield put(actions.saveCluster.failure(error));
        Raven.captureException(error);
    }
    return null;
}

function* deleteClusters({ clusterIds }) {
    try {
        yield call(service.deleteClusters, clusterIds);
        yield fork(getClusters);
    } catch (error) {
        if (error.response) {
            yield put(notificationActions.addNotification(error.response.data.error));
            yield put(notificationActions.removeOldestNotification());
        }
        Raven.captureException(error);
    }
}

function* downloadClusterYaml() {
    try {
        const clusterId = yield select(selectors.getWizardClusterId);
        yield call(service.downloadClusterYaml, clusterId);
    } catch (error) {
        yield put(notificationActions.addNotification('Error while downloading a file'));
        yield put(notificationActions.removeOldestNotification());
        Raven.captureException(error);
    }
}

function* watchFetchRequest() {
    yield takeLatest(types.FETCH_CLUSTERS.REQUEST, getClusters);
}

function* watchDeleteRequest() {
    yield takeLatest(types.DELETE_CLUSTERS, deleteClusters);
}

function* pollCluster(clusterId) {
    while (true) {
        yield call(getCluster, clusterId);
        yield delay(3000); // every 3 sec
    }
}

function* watchDownloadRequest() {
    yield takeLatest(types.DOWNLOAD_CLUSTER_YAML, downloadClusterYaml);
}

function* watchWizard() {
    while (true) {
        const action = yield take([types.NEXT_WIZARD_PAGE, types.PREV_WIZARD_PAGE]);
        const currentPage = yield select(selectors.getWizardCurrentPage);
        const clusterId = yield select(selectors.getWizardClusterId);

        if (action.type === types.NEXT_WIZARD_PAGE && currentPage === wizardPages.FORM) {
            const formData = yield select(getFormValues(clusterFormId));
            const savedClusterId = yield call(saveCluster, { id: clusterId, ...formData });
            yield fork(getClusters);
            if (savedClusterId) {
                yield put(actions.updateWizardState(wizardPages.DEPLOYMENT, savedClusterId));
                yield race([
                    call(pollCluster, savedClusterId),
                    take([types.FINISH_WIZARD, types.PREV_WIZARD_PAGE]),
                ]);
            }
        } else if (
            action.type === types.PREV_WIZARD_PAGE &&
            currentPage === wizardPages.DEPLOYMENT
        ) {
            yield put(actions.updateWizardState(wizardPages.FORM, clusterId));
        }
    }
}

export default function* clusters() {
    yield all([
        fork(watchFetchRequest),
        fork(watchDeleteRequest),
        fork(watchWizard),
        fork(watchDownloadRequest),
    ]);
}
