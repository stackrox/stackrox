import { all, call, fork, put, take } from 'redux-saga/effects';

import { integrationsPath } from 'routePaths';
import { fetchIntegration as serviceFetchIntegration } from 'services/IntegrationsService';
import { actions, types } from 'reducers/integrations';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';

function* fetchIntegrationWrapper(source, action) {
    try {
        const result = yield call(serviceFetchIntegration, source);
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

export default function* integrations() {
    yield all([fork(watchLocation), fork(watchFetchRequest)]);
}
