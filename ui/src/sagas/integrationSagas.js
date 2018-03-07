import { all, take, call, fork, put } from 'redux-saga/effects';

import { fetchNotifiers, fetchRegistries, fetchScanners } from 'services/IntegrationsService';
import { actions, types } from 'reducers/integrations';
import { types as locationActionTypes } from 'reducers/routes';

const integrationsPath = '/main/integrations';

export function* getNotifiers() {
    try {
        const result = yield call(fetchNotifiers);
        yield put(actions.fetchNotifiers.success(result.response));
    } catch (error) {
        yield put(actions.fetchNotifiers.failure(error));
    }
}

export function* getRegistries() {
    try {
        const result = yield call(fetchRegistries);
        yield put(actions.fetchRegistries.success(result.response));
    } catch (error) {
        yield put(actions.fetchRegistries.failure(error));
    }
}

export function* getScanners() {
    try {
        const result = yield call(fetchScanners);
        yield put(actions.fetchScanners.success(result.response));
    } catch (error) {
        yield put(actions.fetchScanners.failure(error));
    }
}

export function* watchIntegrationsLocation() {
    while (true) {
        const action = yield take(locationActionTypes.LOCATION_CHANGE);
        const { payload: location } = action;

        if (location && location.pathname && location.pathname.startsWith(integrationsPath)) {
            yield all([fork(getNotifiers), fork(getRegistries), fork(getScanners)]);
        }
    }
}

export function* watchFetchRequest() {
    while (true) {
        const action = yield take([
            types.FETCH_NOTIFIERS.REQUEST,
            types.FETCH_REGISTRIES.REQUEST,
            types.FETCH_SCANNERS.REQUEST
        ]);
        switch (action.type) {
            case types.FETCH_NOTIFIERS.REQUEST:
                yield fork(getNotifiers);
                break;
            case types.FETCH_REGISTRIES.REQUEST:
                yield fork(getRegistries);
                break;
            case types.FETCH_SCANNERS.REQUEST:
                yield fork(getScanners);
                break;
            default:
                throw new Error(`Unknown action type ${action.type}`);
        }
    }
}

export default function* integrations() {
    yield all([fork(watchIntegrationsLocation), fork(watchFetchRequest)]);
}
