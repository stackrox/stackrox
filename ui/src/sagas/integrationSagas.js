import { all, take, call, fork, put } from 'redux-saga/effects';

import { integrationsPath, policiesPath } from 'routePaths';
import { fetchIntegration } from 'services/IntegrationsService';
import { actions, types } from 'reducers/integrations';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';

function* getNotifiers() {
    try {
        const result = yield call(fetchIntegration, ['notifiers']);
        yield put(actions.fetchNotifiers.success(result.response));
    } catch (error) {
        yield put(actions.fetchNotifiers.failure(error));
    }
}

function* getImageIntegrations() {
    try {
        const result = yield call(fetchIntegration, ['imageIntegrations']);
        yield put(actions.fetchImageIntegrations.success(result.response));
    } catch (error) {
        yield put(actions.fetchImageIntegrations.failure(error));
    }
}

function* watchLocation() {
    const effects = [integrationsPath, policiesPath].map(path =>
        takeEveryNewlyMatchedLocation(path, getNotifiers)
    );
    yield all(
        effects.concat(takeEveryNewlyMatchedLocation(integrationsPath, getImageIntegrations))
    );
}

function* watchFetchRequest() {
    while (true) {
        const action = yield take([
            types.FETCH_NOTIFIERS.REQUEST,
            types.FETCH_IMAGE_INTEGRATIONS.REQUEST
        ]);
        switch (action.type) {
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

export default function* integrations() {
    yield all([fork(watchLocation), fork(watchFetchRequest)]);
}
