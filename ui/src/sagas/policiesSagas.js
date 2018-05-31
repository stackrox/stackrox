import { all, take, call, fork, put, takeLatest, select } from 'redux-saga/effects';
import { push } from 'react-router-redux';

import watchLocation from 'utils/watchLocation';
import * as service from 'services/PoliciesService';
import { actions, types } from 'reducers/policies';
import { actions as notificationActions } from 'reducers/notifications';
import { selectors } from 'reducers';
import searchOptionsToQuery from 'services/searchOptionsToQuery';

const policiesPath = '/main/policies';
const violationsPath = '/main/violations';

export function* getPolicies(filters) {
    try {
        const result = yield call(service.fetchPolicies, filters);
        yield put(actions.fetchPolicies.success(result.response));
    } catch (error) {
        yield put(actions.fetchPolicies.failure(error));
    }
}

export function* getPolicyCategories() {
    try {
        const result = yield call(service.fetchPolicyCategories);
        yield put(actions.fetchPolicyCategories.success(result.response));
    } catch (error) {
        yield put(actions.fetchPolicyCategories.failure(error));
    }
}

export function* filterPoliciesPageBySearch() {
    const searchOptions = yield select(selectors.getPoliciesSearchOptions);
    const filters = {
        query: searchOptionsToQuery(searchOptions)
    };
    yield fork(getPolicies, filters);
}

export function* loadViolationsPage() {
    yield fork(getPolicies, {});
}

function* createPolicy(policy) {
    try {
        const { data } = yield call(service.createPolicy, policy);
        yield put(actions.setPolicyWizardState({ current: '', isNew: false }));
        yield put(push(`/main/policies/${data.id}`));
    } catch (error) {
        console.error(error);
        if (error.response) {
            yield put(notificationActions.addNotification(error.response.data.error));
            yield put(notificationActions.removeOldestNotification());
        }
        yield put(actions.setPolicyWizardState({ current: 'PREVIEW', policy }));
    }
}

function* savePolicy(policy) {
    try {
        yield call(service.savePolicy, policy);
        yield put(actions.setPolicyWizardState({ current: '', isNew: false }));
        yield fork(getPolicies);
    } catch (error) {
        console.error(error);
        if (error.response) {
            yield put(notificationActions.addNotification(error.response.data.error));
            yield put(notificationActions.removeOldestNotification());
        }
        yield put(actions.setPolicyWizardState({ current: 'PREVIEW', policy }));
    }
}

function* updatePolicy(action) {
    try {
        yield call(service.savePolicy, action.policy);
        yield fork(getPolicies);
    } catch (error) {
        console.error(error);
        if (error.response) {
            yield put(notificationActions.addNotification(error.response.data.error));
            yield put(notificationActions.removeOldestNotification());
        }
    }
}

function* reassessPolicies() {
    try {
        yield call(service.reassessPolicies);
        yield put(notificationActions.addNotification('Policies were reassessed'));
        yield put(notificationActions.removeOldestNotification());
    } catch (error) {
        console.error(error);
        if (error.response) {
            yield put(notificationActions.addNotification(error.response.data.error));
            yield put(notificationActions.removeOldestNotification());
        }
    }
}

function* getDryRun(policy) {
    try {
        const policyDryRun = yield call(service.getDryRun, policy);
        yield put(
            actions.setPolicyWizardState({
                current: 'PREVIEW',
                dryrun: policyDryRun.data,
                policy
            })
        );
    } catch (error) {
        console.error(error);
        if (error.response) {
            yield put(notificationActions.addNotification(error.response.data.error));
            yield put(notificationActions.removeOldestNotification());
        }
    }
}

export function* loadPoliciesPage() {
    yield all([fork(filterPoliciesPageBySearch), fork(getPolicyCategories)]);
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

function* watchUpdateRequest() {
    yield takeLatest(types.UPDATE_POLICY, updatePolicy);
}

function* watchReassessPolicies() {
    yield takeLatest(types.REASSESS_POLICIES, reassessPolicies);
}

function* watchWizardState() {
    while (true) {
        const action = yield take(types.SET_POLICY_WIZARD_STATE);
        switch (action.state.current) {
            case 'PRE_PREVIEW':
                yield fork(getDryRun, action.state.policy);
                break;
            case 'SAVE':
                yield fork(savePolicy, action.state.policy);
                break;
            case 'CREATE':
                yield fork(createPolicy, action.state.policy);
                break;
            default:
                break;
        }
    }
}

export default function* policies() {
    yield all([
        fork(watchLocation, policiesPath, loadPoliciesPage),
        fork(watchLocation, violationsPath, loadViolationsPage),
        fork(watchFetchRequest),
        fork(watchWizardState),
        fork(watchReassessPolicies),
        fork(watchUpdateRequest),
        fork(watchPoliciesSearchOptions)
    ]);
}
