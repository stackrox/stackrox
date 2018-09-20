import { all, take, takeEvery, call, fork, put, takeLatest, select } from 'redux-saga/effects';
import { push } from 'react-router-redux';
import Raven from 'raven-js';

import { policiesPath, violationsPath } from 'routePaths';
import * as service from 'services/PoliciesService';
import { actions, types } from 'reducers/policies';
import { actions as notificationActions } from 'reducers/notifications';
import { selectors } from 'reducers';
import searchOptionsToQuery from 'services/searchOptionsToQuery';
import { takeEveryNewlyMatchedLocation, takeEveryLocation } from 'utils/sagaEffects';

export function* getPolicies(filters) {
    try {
        const result = yield call(service.fetchPolicies, filters);
        yield put(actions.fetchPolicies.success(result.response));
    } catch (error) {
        yield put(actions.fetchPolicies.failure(error));
    }
}

export function* getPolicy(policyId) {
    yield put(actions.fetchPolicy.request());
    try {
        const result = yield call(service.fetchPolicy, policyId);
        yield put(actions.fetchPolicy.success(result.response));
    } catch (error) {
        yield put(actions.fetchPolicy.failure(error));
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
    if (searchOptions.length && searchOptions[searchOptions.length - 1].type) {
        return;
    }
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
        yield fork(filterPoliciesPageBySearch);
    } catch (error) {
        if (error.response) {
            yield put(notificationActions.addNotification(error.response.data.error));
            yield put(notificationActions.removeOldestNotification());
        } else {
            // TODO-ivan: use global user notification system to display the problem to the user as well
            Raven.captureException(error);
        }
        yield put(actions.setPolicyWizardState({ current: 'PREVIEW', policy }));
    }
}

function* savePolicy(policy) {
    try {
        yield call(service.savePolicy, policy);
        yield fork(getPolicy, policy.id);
        yield put(actions.setPolicyWizardState({ current: '', isNew: false }));
        yield fork(filterPoliciesPageBySearch);
    } catch (error) {
        if (error.response) {
            yield put(notificationActions.addNotification(error.response.data.error));
            yield put(notificationActions.removeOldestNotification());
        } else {
            // TODO-ivan: use global user notification system to display the problem to the user as well
            Raven.captureException(error);
        }
        yield put(actions.setPolicyWizardState({ current: 'PREVIEW', policy }));
    }
}

function* deletePolicies({ policyIds }) {
    try {
        yield call(service.deletePolicies, policyIds);
        yield fork(filterPoliciesPageBySearch);
    } catch (error) {
        // TODO-ivan: use global user notification system to display the problem to the user as well
        Raven.captureException(error);
    }
}

function* updatePolicy(action) {
    try {
        yield call(service.savePolicy, action.policy);
        yield fork(filterPoliciesPageBySearch);
    } catch (error) {
        if (error.response) {
            yield put(notificationActions.addNotification(error.response.data.error));
            yield put(notificationActions.removeOldestNotification());
        } else {
            // TODO-ivan: use global user notification system to display the problem to the user as well
            Raven.captureException(error);
        }
    }
}

function* reassessPolicies() {
    try {
        yield call(service.reassessPolicies);
        yield put(notificationActions.addNotification('Policies were reassessed'));
        yield put(notificationActions.removeOldestNotification());
    } catch (error) {
        if (error.response) {
            yield put(notificationActions.addNotification(error.response.data.error));
            yield put(notificationActions.removeOldestNotification());
        } else {
            // TODO-ivan: use global user notification system to display the problem to the user as well
            Raven.captureException(error);
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
        if (error.response) {
            yield put(notificationActions.addNotification(error.response.data.error));
            yield put(notificationActions.removeOldestNotification());
        } else {
            // TODO-ivan: use global user notification system to display the problem to the user as well
            Raven.captureException(error);
        }
    }
}

export function* loadPoliciesPage() {
    yield all([fork(filterPoliciesPageBySearch), fork(getPolicyCategories)]);
}

export function* loadPolicy({ match }) {
    const { policyId } = match.params;
    if (policyId) {
        yield fork(getPolicy, policyId);
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

export function* watchFetchRequest() {
    while (true) {
        const action = yield take(types.FETCH_POLICIES.REQUEST);
        if (action.type === types.FETCH_POLICIES.REQUEST) {
            yield fork(filterPoliciesPageBySearch);
        }
    }
}

function* watchDeletePolicies() {
    yield takeLatest(types.DELETE_POLICIES, deletePolicies);
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

function* updatePolicyDisabledState({ policyId, disabled }) {
    try {
        yield call(service.updatePolicyDisabledState, policyId, disabled);
        yield fork(filterPoliciesPageBySearch);
    } catch (error) {
        // TODO-ivan: use global user notification system to display the problem to the user as well
        Raven.captureException(error);
    }
}

export default function* policies() {
    yield all([
        takeEveryNewlyMatchedLocation(policiesPath, loadPoliciesPage),
        takeEveryLocation(policiesPath, loadPolicy),
        takeEveryNewlyMatchedLocation(violationsPath, loadViolationsPage),
        fork(watchFetchRequest),
        fork(watchWizardState),
        fork(watchReassessPolicies),
        fork(watchDeletePolicies),
        fork(watchUpdateRequest),
        fork(watchPoliciesSearchOptions),
        takeEvery(types.UPDATE_POLICY_DISABLED_STATE, updatePolicyDisabledState)
    ]);
}
