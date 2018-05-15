import { all, take, takeLatest, call, fork, put, select } from 'redux-saga/effects';

import { fetchPolicies, fetchPolicyCategories } from 'services/PoliciesService';
import { actions, types } from 'reducers/policies';
import { types as locationActionTypes } from 'reducers/routes';
import { selectors } from 'reducers';
import searchOptionsToQuery from 'services/searchOptionsToQuery';

const policiesPath = '/main/policies';

export function* getPolicies(filters) {
    try {
        const result = yield call(fetchPolicies, filters);
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

function* filterPoliciesPageBySearch() {
    const searchOptions = yield select(selectors.getPoliciesSearchOptions);
    const filters = {
        query: searchOptionsToQuery(searchOptions)
    };
    yield fork(getPolicies, filters);
}

export function* watchLocation() {
    while (true) {
        const action = yield take(locationActionTypes.LOCATION_CHANGE);
        const { payload: location } = action;
        const { pathname } = location;

        if (location && pathname && pathname.startsWith(policiesPath)) {
            yield all([fork(filterPoliciesPageBySearch), fork(getPolicyCategories)]);
        }
    }
}

export function* watchFetchRequest() {
    while (true) {
        const action = yield take(types.FETCH_POLICIES.REQUEST);
        if (action.type === types.FETCH_POLICIES.REQUEST) {
            yield fork(filterPoliciesPageBySearch);
        }
    }
}

function* watchPoliciesSearchOptions() {
    yield takeLatest(types.SET_SEARCH_OPTIONS, filterPoliciesPageBySearch);
}

export default function* policies() {
    yield all([fork(watchLocation), fork(watchFetchRequest), fork(watchPoliciesSearchOptions)]);
}
