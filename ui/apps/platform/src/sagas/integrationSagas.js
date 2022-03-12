import { all, take, call, fork, put, takeLatest } from 'redux-saga/effects';
import Raven from 'raven-js';

import { integrationsPath, policiesPath, networkPath } from 'routePaths';
import * as service from 'services/IntegrationsService';
import * as AuthService from 'services/AuthService';
import * as BackupIntegrationsService from 'services/BackupIntegrationsService';
import { actions as clusterActions } from 'reducers/clusters';
import { actions, types } from 'reducers/integrations';
import { actions as notificationActions } from 'reducers/notifications';
import { actions as authActions } from 'reducers/auth';
import { actions as apiTokenActions } from 'reducers/apitokens';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';

const fetchIntegrationsActionMap = {
    authPlugins: actions.fetchAuthPlugins.request(),
    authProviders: authActions.fetchAuthProviders.request(),
    backups: actions.fetchBackups.request(),
    imageIntegrations: actions.fetchImageIntegrations.request(),
    signatureIntegrations: actions.fetchSignatureIntegrations.request(),
    notifiers: actions.fetchNotifiers.request(),
    clusters: clusterActions.fetchClusters.request(),
    apitoken: apiTokenActions.fetchAPITokens.request(),
};

function getFriendlyErrorMessage(type, response) {
    let errorMessage = '';

    switch (type) {
        case 'awsSecurityHub': {
            if (response?.data?.error?.includes('403')) {
                errorMessage = 'Credentials are incorrect. Please re-enter them and try again.';
            } else if (response?.data?.error?.includes('AccessDenied')) {
                errorMessage =
                    'Access denied. The account number does not match the credentials provided. Please re-enter the account number and try again.';
            } else if (response?.data?.error?.includes('not subscribed')) {
                errorMessage =
                    'Chosen region is not subscribed to StackRox Security Hub integration. Please subscribe through AWS Console, or choose a region that is subscribed.';
            } else if (
                response?.data?.error?.includes('InvalidSignature') ||
                response?.data?.error?.includes('UnrecognizedClient')
            ) {
                errorMessage = '403 Access Denied. Please check your inputs and try again.';
            } else {
                errorMessage =
                    'An error has occurred. Please check the central logs for more information.';
            }
            break;
        }
        default: {
            errorMessage = response?.data?.error || 'An unknown error has occurred.';
        }
    }

    return errorMessage;
}

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

function* getAuthPlugins() {
    yield call(fetchIntegrationWrapper, 'authPlugins', actions.fetchAuthPlugins);
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
    const effects = [
        getImageIntegrations,
        getSignatureIntegrations,
        getNotifiers,
        getBackups,
        getAuthPlugins,
    ].map((fetchFunc) => takeEveryNewlyMatchedLocation(integrationsPath, fetchFunc));
    yield all([
        ...effects,
        takeEveryNewlyMatchedLocation(policiesPath, getNotifiers),
        takeEveryNewlyMatchedLocation(networkPath, getNotifiers),
    ]);
}

function* watchFetchRequest() {
    while (true) {
        const action = yield take([
            types.FETCH_AUTH_PLUGINS.REQUEST,
            types.FETCH_BACKUPS.REQUEST,
            types.FETCH_IMAGE_INTEGRATIONS.REQUEST,
            types.FETCH_SIGNATURE_INTEGRATIONS.REQUEST,
            types.FETCH_NOTIFIERS.REQUEST,
        ]);
        switch (action.type) {
            case types.FETCH_AUTH_PLUGINS.REQUEST:
                yield fork(getAuthPlugins);
                break;
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

function* saveIntegration(action) {
    const { source, sourceType, integration, options, displayName } = action.params;
    try {
        if (source === 'authProviders') {
            yield call(AuthService.saveAuthProvider, integration);
            if (sourceType === 'apitoken') {
                yield put(fetchIntegrationsActionMap[sourceType]);
            } else {
                yield put(fetchIntegrationsActionMap[source]);
            }
        } else {
            if (integration.id) {
                yield call(service.saveIntegration, source, integration, options);
            } else {
                yield call(service.createIntegration, source, integration);
            }
            yield put(fetchIntegrationsActionMap[source]);
        }
        yield put(
            notificationActions.addNotification(
                `Successfully integrated ${displayName || integration.type}`
            )
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

function* testIntegration(action) {
    const { source, integration, options } = action;
    try {
        yield call(service.testIntegration, source, integration, options);
        yield put(notificationActions.addNotification('Integration test was successful'));
        yield put(notificationActions.removeOldestNotification());
    } catch (error) {
        if (error.response) {
            const errorMessage = getFriendlyErrorMessage(integration?.type, error?.response);

            yield put(notificationActions.addNotification(errorMessage));
            yield put(notificationActions.removeOldestNotification());
        } else {
            Raven.captureException(error);
        }
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
        fork(watchBackupRequest),
    ]);
}
