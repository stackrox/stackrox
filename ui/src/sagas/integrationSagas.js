import { all, take, call, fork, put } from 'redux-saga/effects';

import { integrationsPath, policiesPath } from 'routePaths';
import { fetchIntegration } from 'services/IntegrationsService';
import { types as clusterTypes } from 'reducers/clusters';
import { actions, types } from 'reducers/integrations';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';

// Call fetchIntegration with the given source, and pass the response/failure
// with the given action type.
function* fetchIntegrationWrapper(source, action) {
    try {
        const result = yield call(fetchIntegration, [source]);
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
    yield all([...effects, takeEveryNewlyMatchedLocation(policiesPath, getNotifiers)]);
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

export default function* integrations() {
    yield all([fork(watchLocation), fork(watchFetchRequest)]);
}
