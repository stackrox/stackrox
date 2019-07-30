import { all, call, fork, put } from 'redux-saga/effects';
import * as service from 'services/FeatureFlagsService';
import { actions } from 'reducers/featureFlags';

function* getFeatureFlags() {
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
