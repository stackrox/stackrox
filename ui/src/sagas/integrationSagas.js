import { all, take, call, fork, put, takeLatest } from 'redux-saga/effects';
import Raven from 'raven-js';

import { integrationsPath, policiesPath, environmentPath } from 'routePaths';
import * as service from 'services/IntegrationsService';
import * as AuthService from 'services/AuthService';
import { actions as clusterActions, types as clusterTypes } from 'reducers/clusters';
import { actions, types } from 'reducers/integrations';
import { actions as notificationActions } from 'reducers/notifications';
import { actions as authActions } from 'reducers/auth';
import { actions as apiTokenActions } from 'reducers/apitokens';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';

const fetchIntegrationsActionMap = {
    authProviders: authActions.fetchAuthProviders.request(),
    dnrIntegrations: actions.fetchDNRIntegrations.request(),
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

function* getNotifiers() {
    yield call(fetchIntegrationWrapper, 'notifiers', actions.fetchNotifiers);
}

function* getDNRIntegrations() {
    yield call(fetchIntegrationWrapper, 'dnrIntegrations', actions.fetchDNRIntegrations);
}

function* getImageIntegrations() {
    yield call(fetchIntegrationWrapper, 'imageIntegrations', actions.fetchImageIntegrations);
}

function* watchLocation() {
    const effects = [getDNRIntegrations, getImageIntegrations, getNotifiers].map(fetchFunc =>
        takeEveryNewlyMatchedLocation(integrationsPath, fetchFunc)
    );
    yield all([
        ...effects,
        takeEveryNewlyMatchedLocation(policiesPath, getNotifiers),
        takeEveryNewlyMatchedLocation(environmentPath, getNotifiers)
    ]);
}

function* watchFetchRequest() {
    while (true) {
        const action = yield take([
            clusterTypes.FETCH_CLUSTERS.SUCCESS,
            types.FETCH_DNR_INTEGRATIONS.REQUEST,
            types.FETCH_IMAGE_INTEGRATIONS.REQUEST,
            types.FETCH_NOTIFIERS.REQUEST
        ]);
        switch (action.type) {
            case types.FETCH_NOTIFIERS.REQUEST:
                yield fork(getNotifiers);
                break;
            case types.FETCH_IMAGE_INTEGRATIONS.REQUEST:
                yield fork(getImageIntegrations);
                break;
            case clusterTypes.FETCH_CLUSTERS.SUCCESS:
            case types.FETCH_DNR_INTEGRATIONS.REQUEST:
                yield fork(getDNRIntegrations);
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
            yield call(AuthService.deleteAuthProviders(ids));
            if (sourceType === 'apitoken') yield put(fetchIntegrationsActionMap[sourceType]);
            else yield put(fetchIntegrationsActionMap[source]);
        } else {
            yield call(service.deleteIntegrations, source, ids);
            yield put(fetchIntegrationsActionMap[source]);
        }
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

function* watchSaveRequest() {
    yield takeLatest(types.SAVE_INTEGRATION.REQUEST, saveIntegration);
}

function* watchTestRequest() {
    yield takeLatest(types.TEST_INTEGRATION, testIntegration);
}

function* watchDeleteRequest() {
    yield takeLatest(types.DELETE_INTEGRATIONS, deleteIntegrations);
}

export default function* integrations() {
    yield all([
        fork(watchLocation),
        fork(watchFetchRequest),
        fork(watchSaveRequest),
        fork(watchTestRequest),
        fork(watchDeleteRequest)
    ]);
}
