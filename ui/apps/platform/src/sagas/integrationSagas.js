import { all, take, call, fork, put, takeLatest } from 'redux-saga/effects';
import Raven from 'raven-js';

import { integrationsPath } from 'routePaths';
import * as service from 'services/IntegrationsService';
import * as AuthService from 'services/AuthService';
import * as BackupIntegrationsService from 'services/BackupIntegrationsService';
import { actions, types } from 'reducers/integrations';
import { actions as notificationActions } from 'reducers/notifications';
import { actions as apiTokenActions } from 'reducers/apitokens';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';

const fetchIntegrationsActionMap = {
    backups: actions.fetchBackups.request(),
    imageIntegrations: actions.fetchImageIntegrations.request(),
    signatureIntegrations: actions.fetchSignatureIntegrations.request(),
    notifiers: actions.fetchNotifiers.request(),
    apitoken: apiTokenActions.fetchAPITokens.request(),
};

// Call fetchIntegration with the given source, and pass the response/failure
// with the given action type.
function* fetchIntegrationWrapper(source, action) {
    try {
        const result = yield call(service.fetchIntegration, source);
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

function* getSignatureIntegrations() {
    yield call(
        fetchIntegrationWrapper,
        'signatureIntegrations',
        actions.fetchSignatureIntegrations
    );
}

function* watchLocation() {
    yield all(
        [getImageIntegrations, getSignatureIntegrations, getNotifiers, getBackups].map(
            (fetchFunc) => takeEveryNewlyMatchedLocation(integrationsPath, fetchFunc)
        )
    );
}

function* watchFetchRequest() {
    while (true) {
        const action = yield take([
            types.FETCH_BACKUPS.REQUEST,
            types.FETCH_IMAGE_INTEGRATIONS.REQUEST,
            types.FETCH_SIGNATURE_INTEGRATIONS.REQUEST,
            types.FETCH_NOTIFIERS.REQUEST,
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
            case types.FETCH_SIGNATURE_INTEGRATIONS.REQUEST:
                yield fork(getSignatureIntegrations);
                break;

            default:
                throw new Error(`Unknown action type ${action.type}`);
        }
    }
}

function* deleteIntegrations({ source, sourceType, ids }) {
    try {
        if (source === 'authProviders') {
            yield call(AuthService.deleteAuthProviders, ids);
            if (sourceType === 'apitoken') {
                yield put(fetchIntegrationsActionMap[sourceType]);
            } else {
                yield put(fetchIntegrationsActionMap[source]);
            }
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

function* triggerBackup(action) {
    const { id } = action;
    try {
        yield call(BackupIntegrationsService.triggerBackup, id);
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
        fork(watchDeleteRequest),
        fork(watchBackupRequest),
    ]);
}
