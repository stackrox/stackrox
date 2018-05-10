import { all, take, call, fork, put } from 'redux-saga/effects';

import { fetchPolicies, fetchPolicyCategories } from 'services/PoliciesService';
import { actions, types } from 'reducers/policies';
import { types as locationActionTypes } from 'reducers/routes';

const policiesPath = '/main/policies';

export function* getPolicies() {
    try {
        const result = yield call(fetchPolicies);
        yield put(actions.fetchPolicies.success(result.response));
    } catch (error) {
        yield put(actions.fetchPolicies.failure(error));
    }
}

export function* getPolicyCategories() {
    try {
        const result = yield call(fetchPolicyCategories);
        yield put(actions.fetchPolicyCategories.success(result.response));
    } catch (error) {
        yield put(actions.fetchPolicyCategories.failure(error));
    }
}

export function* watchLocation() {
    while (true) {
        const action = yield take(locationActionTypes.LOCATION_CHANGE);
        const { payload: location } = action;
        const { pathname } = location;

        if (location && pathname && pathname.startsWith(policiesPath)) {
            yield all([fork(getPolicies), fork(getPolicyCategories)]);
        }
    }
}

export function* watchFetchRequest() {
    while (true) {
        const action = yield take(types.FETCH_POLICIES.REQUEST);
        if (action.type === types.FETCH_POLICIES.REQUEST) {
            yield fork(getPolicies);
        }
    }
}

export default function* policies() {
    yield all([fork(watchLocation), fork(watchFetchRequest)]);
}
