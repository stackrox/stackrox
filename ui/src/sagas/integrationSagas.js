import { all, take, call, fork, put, takeLatest } from 'redux-saga/effects';
import Raven from 'raven-js';

import { integrationsPath, policiesPath, networkPath } from 'routePaths';
import * as service from 'services/IntegrationsService';
import * as AuthService from 'services/AuthService';
import { actions as clusterActions } from 'reducers/clusters';
import { actions, types } from 'reducers/integrations';
import { actions as notificationActions } from 'reducers/notifications';
import { actions as authActions } from 'reducers/auth';
import { actions as apiTokenActions } from 'reducers/apitokens';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';

const fetchIntegrationsActionMap = {
    authProviders: authActions.fetchAuthProviders.request(),
    backups: actions.fetchBackups.request(),
    imageIntegrations: actions.fetchImageIntegrations.request(),
    notifiers: actions.fetchNotifiers.request(),
    clusters: clusterActions.fetchClusters.request(),
    apitoken: apiTokenActions.fetchAPITokens.request()
};

// Call fetchIntegration with the given source, and pass the response/failure
// with the given action type.
function* fetchIntegrationWrapper(source, action) {
    try {
        const result = yield call(service.fetchIntegration, [source]);
        yield put(action.success(result.response));
    } catch (error) {
        yield put(action.failure(error));
    }
}

function* getBackups() {
    yield call(fetchIntegrationWrapper, 'backups', actions.fetchBackups);
}

function* getNotifiers() {
    yield call(fetchIntegrationWrapper, 'notifiers', actions.fetchNotifiers);
}

function* getImageIntegrations() {
    yield call(fetchIntegrationWrapper, 'imageIntegrations', actions.fetchImageIntegrations);
}

function* watchLocation() {
    const effects = [getImageIntegrations, getNotifiers, getBackups].map(fetchFunc =>
        takeEveryNewlyMatchedLocation(integrationsPath, fetchFunc)
    );
    yield all([
        ...effects,
        takeEveryNewlyMatchedLocation(policiesPath, getNotifiers),
        takeEveryNewlyMatchedLocation(networkPath, getNotifiers)
    ]);
}

function* watchFetchRequest() {
    while (true) {
        const action = yield take([
            types.FETCH_BACKUPS.REQUEST,
            types.FETCH_IMAGE_INTEGRATIONS.REQUEST,
            types.FETCH_NOTIFIERS.REQUEST
        ]);
        switch (action.type) {
            case types.FETCH_BACKUPS.REQUEST:
                yield fork(getBackups);
                break;
            case types.FETCH_NOTIFIERS.REQUEST:
                yield fork(getNotifiers);
                break;
            case types.FETCH_IMAGE_INTEGRATIONS.REQUEST:
                yield fork(getImageIntegrations);
                break;

            default:
                throw new Error(`Unknown action type ${action.type}`);
        }
    }
}

function* saveIntegration(action) {
    const { source, sourceType, integration } = action.params;
    try {
        if (source === 'authProviders') {
            yield call(AuthService.saveAuthProvider, integration);
            if (sourceType === 'apitoken') yield put(fetchIntegrationsActionMap[sourceType]);
            else yield put(fetchIntegrationsActionMap[source]);
        } else {
            if (integration.id) yield call(service.saveIntegration, source, integration);
            else yield call(service.createIntegration, source, integration);
            yield put(fetchIntegrationsActionMap[source]);
        }
        yield put(
            notificationActions.addNotification(`Successfully integrated ${integration.type}`)
        );
        yield put(notificationActions.removeOldestNotification());
        yield put(actions.setCreateState(false));
    } catch (error) {
        if (error.response) {
            yield put(notificationActions.addNotification(error.response.data.error));
            yield put(notificationActions.removeOldestNotification());
        } else {
            Raven.captureException(error);
        }
    }
}

function* deleteIntegrations({ source, sourceType, ids }) {
    try {
        if (source === 'authProviders') {
            yield call(AuthService.deleteAuthProviders, ids);
            if (sourceType === 'apitoken') yield put(fetchIntegrationsActionMap[sourceType]);
            else yield put(fetchIntegrationsActionMap[source]);
        } else {
            yield call(service.deleteIntegrations, source, ids);
            yield put(fetchIntegrationsActionMap[source]);
        }
        const toastMessage = `Successfully deleted ${ids.length} integration${
            ids.length === 1 ? '' : 's'
        }`;
        yield put(notificationActions.addNotification(toastMessage));
        yield put(notificationActions.removeOldestNotification());
    } catch (error) {
        Raven.captureException(error);
    }
}

function* testIntegration(action) {
    const { source, integration } = action;
    try {
        yield call(service.testIntegration, source, integration);
        yield put(notificationActions.addNotification('Integration test was successful'));
        yield put(notificationActions.removeOldestNotification());
    } catch (error) {
        if (error.response) {
            yield put(notificationActions.addNotification(error.response.data.error));
            yield put(notificationActions.removeOldestNotification());
        } else {
            Raven.captureException(error);
        }
    }
}

function* triggerBackup(action) {
    const { id } = action;
    try {
        yield call(service.triggerBackup, id);
        yield put(notificationActions.addNotification('Backup was successful'));
        yield put(notificationActions.removeOldestNotification());
    } catch (error) {
        if (error.response) {
            yield put(notificationActions.addNotification(error.response.data.error));
            yield put(notificationActions.removeOldestNotification());
        } else {
            Raven.captureException(error);
        }
    }
}

function* watchSaveRequest() {
    yield takeLatest(types.SAVE_INTEGRATION.REQUEST, saveIntegration);
}

function* watchTestRequest() {
    yield takeLatest(types.TEST_INTEGRATION, testIntegration);
}

function* watchDeleteRequest() {
    yield takeLatest(types.DELETE_INTEGRATIONS, deleteIntegrations);
}

function* watchBackupRequest() {
    yield takeLatest(types.TRIGGER_BACKUP, triggerBackup);
}

export default function* integrations() {
    yield all([
        fork(watchLocation),
        fork(watchFetchRequest),
        fork(watchSaveRequest),
        fork(watchTestRequest),
        fork(watchDeleteRequest),
        fork(watchBackupRequest)
    ]);
}
