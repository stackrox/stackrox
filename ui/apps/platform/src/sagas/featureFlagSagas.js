import { all, call, fork, put } from 'redux-saga/effects';
import * as service from 'services/FeatureFlagsService';
import { actions } from 'reducers/featureFlags';

function* getFeatureFlags() {
    /*
     * Call request because featureFlags.isLoading reducer needs the action
     * for subsequent requests (for example, manual refresh; or log out, and then log in again).
     * Imitate request-success-failure pattern in redux-thunk.
     * In this case, redux-saga makes the request independently of the action.
     */
    yield put(actions.fetchFeatureFlags.request());
    try {
        const result = yield call(service.fetchFeatureFlags);
        yield put(actions.fetchFeatureFlags.success(result.response));
    } catch (error) {
        yield put(actions.fetchFeatureFlags.failure(error));
    }
}

export default function* featureFlags() {
    yield all([fork(getFeatureFlags)]);
}
