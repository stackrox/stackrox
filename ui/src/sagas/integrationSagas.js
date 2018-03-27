import { all, take, call, fork, put } from 'redux-saga/effects';

import { fetchIntegration } from 'services/IntegrationsService';
import { actions, types } from 'reducers/integrations';
import { types as locationActionTypes } from 'reducers/routes';

const integrationsPath = '/main/integrations';

export function* getNotifiers() {
    try {
        const result = yield call(fetchIntegration, ['notifiers']);
        yield put(actions.fetchNotifiers.success(result.response));
    } catch (error) {
        yield put(actions.fetchNotifiers.failure(error));
    }
}

export function* getImageIntegrations() {
    try {
        const result = yield call(fetchIntegration, ['imageIntegrations']);
        yield put(actions.fetchImageIntegrations.success(result.response));
    } catch (error) {
        yield put(actions.fetchImageIntegrations.failure(error));
    }
}

export function* watchIntegrationsLocation() {
    while (true) {
        const action = yield take(locationActionTypes.LOCATION_CHANGE);
        const { payload: location } = action;

        if (location && location.pathname && location.pathname.startsWith(integrationsPath)) {
            yield all([fork(getNotifiers), fork(getImageIntegrations)]);
        }
    }
}

export function* watchFetchRequest() {
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
    yield all([fork(watchIntegrationsLocation), fork(watchFetchRequest)]);
}
