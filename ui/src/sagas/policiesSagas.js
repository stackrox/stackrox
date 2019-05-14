import { all, take, takeEvery, call, fork, put, takeLatest, select } from 'redux-saga/effects';
import { push } from 'react-router-redux';
import Raven from 'raven-js';

import { policiesPath, violationsPath } from 'routePaths';
import * as service from 'services/PoliciesService';
import { actions as backendActions, types as backendTypes } from 'reducers/policies/backend';
import { actions as pageActions } from 'reducers/policies/page';
import { types as searchTypes } from 'reducers/policies/search';
import { actions as tableActions, types as tableTypes } from 'reducers/policies/table';
import { actions as wizardActions, types as wizardTypes } from 'reducers/policies/wizard';
import { actions as notificationActions } from 'reducers/notifications';
import { selectors } from 'reducers';
import searchOptionsToQuery from 'services/searchOptionsToQuery';
import { takeEveryNewlyMatchedLocation, takeEveryLocation } from 'utils/sagaEffects';

import wizardStages from 'Containers/Policies/Wizard/wizardStages';

export function* getPolicies(filters) {
    try {
        const result = yield call(service.fetchPolicies, filters);
        yield put(backendActions.fetchPolicies.success(result.response));

        // If the fetched policies do not contain the wizard policy, close the wizard.
        const fetchedPolicIds = result.response.result.policies;
        if (fetchedPolicIds) {
            const policy = yield select(selectors.getWizardPolicy);
            if (policy && fetchedPolicIds.find(id => id === policy.id) === undefined) {
                yield put(pageActions.closeWizard());
            }
        }
    } catch (error) {
        yield put(backendActions.fetchPolicies.failure(error));
    }
}

export function* getPolicyCategories() {
    try {
        const result = yield call(service.fetchPolicyCategories);
        yield put(backendActions.fetchPolicyCategories.success(result.response));
    } catch (error) {
        yield put(backendActions.fetchPolicyCategories.failure(error));
    }
}

export function* getPolicy(policyId) {
    yield put(backendActions.fetchPolicy.request());
    try {
        const [policyResult] = yield all([
            call(service.fetchPolicy, policyId),
            call(getPolicyCategories) // make sure we have latest categories for the wizard
        ]);
        yield put(backendActions.fetchPolicy.success(policyResult.response));

        // When a policy is selected, make sure the wizard is opened for it.
        const fetchedPolicies = Object.values(policyResult.response.entities.policy);
        if (fetchedPolicies.length === 1) {
            yield put(tableActions.selectPolicyId(fetchedPolicies[0].id));
            yield put(wizardActions.setWizardPolicy(fetchedPolicies[0]));
            yield put(wizardActions.setWizardStage(wizardStages.details));
            yield put(pageActions.openWizard());
        }
    } catch (error) {
        yield put(backendActions.fetchPolicy.failure(error));
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
        yield put(wizardActions.setWizardStage(wizardStages.details));
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
        yield put(wizardActions.setWizardPolicy(policy));
        yield put(wizardActions.setWizardStage(wizardStages.enforcement));
    }
}

function* savePolicy(policy) {
    try {
        yield call(service.savePolicy, policy);
        yield fork(getPolicy, policy.id);
        yield put(wizardActions.setWizardStage(wizardStages.details));
        yield fork(filterPoliciesPageBySearch);
    } catch (error) {
        if (error.response) {
            yield put(notificationActions.addNotification(error.response.data.error));
            yield put(notificationActions.removeOldestNotification());
        } else {
            // TODO-ivan: use global user notification system to display the problem to the user as well
            Raven.captureException(error);
        }
        yield put(wizardActions.setWizardPolicy(policy));
        yield put(wizardActions.setWizardStage(wizardStages.enforcement));
    }
}

function* deletePolicies({ policyIds }) {
    try {
        yield call(service.deletePolicies, policyIds);
        const successToastMessage = `Successfully deleted ${
            policyIds.length === 1 ? 'policy' : 'policies'
        }`;
        yield put(notificationActions.addNotification(successToastMessage));
        yield put(notificationActions.removeOldestNotification());
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

function* enablePoliciesNotification({ policyIds, notifierIds }) {
    try {
        yield call(service.enablePoliciesNotification, policyIds, notifierIds);
        const successToastMessage = `Successfully enabled ${
            policyIds.length === 1 ? 'policy' : 'policies'
        } notification`;
        yield put(notificationActions.addNotification(successToastMessage));
        yield put(notificationActions.removeOldestNotification());
        yield fork(filterPoliciesPageBySearch);
    } catch (error) {
        // TODO-ivan: use global user notification system to display the problem to the user as well
        Raven.captureException(error);
    }
}

function* getDryRun(policy) {
    try {
        const policyDryRun = yield call(service.getDryRun, policy);
        yield put(wizardActions.setWizardDryRun(policyDryRun.data));
        yield put(wizardActions.setWizardStage(wizardStages.preview));
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
    yield takeLatest(searchTypes.SET_SEARCH_OPTIONS, filterPoliciesPageBySearch);
}

function* watchUpdateRequest() {
    yield takeLatest(backendTypes.UPDATE_POLICY, updatePolicy);
}

function* watchReassessPolicies() {
    yield takeLatest(backendTypes.REASSESS_POLICIES, reassessPolicies);
}

export function* watchFetchRequest() {
    while (true) {
        const action = yield take(backendTypes.FETCH_POLICIES.REQUEST);
        if (action.type === backendTypes.FETCH_POLICIES.REQUEST) {
            yield fork(filterPoliciesPageBySearch);
        }
    }
}

function* watchDeletePolicies() {
    yield takeLatest(backendTypes.DELETE_POLICIES, deletePolicies);
}

function* watchEnablePoliciesNotification() {
    yield takeLatest(backendTypes.ENABLE_POLICIES_NOTIFICATION, enablePoliciesNotification);
}

function* watchWizardState() {
    while (true) {
        const action = yield take(wizardTypes.SET_WIZARD_STAGE);
        const policy = yield select(selectors.getWizardPolicy);
        switch (action.stage) {
            case wizardStages.prepreview:
                yield fork(getDryRun, policy);
                break;
            case wizardStages.save:
                yield fork(savePolicy, policy);
                break;
            case wizardStages.create:
                yield fork(createPolicy, policy);
                break;
            default:
                break;
        }
    }
}

function* updatePolicyDisabled({ policyId, disabled }) {
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
        fork(watchEnablePoliciesNotification),
        fork(watchUpdateRequest),
        fork(watchPoliciesSearchOptions),
        takeEvery(tableTypes.UPDATE_POLICY_DISABLED_STATE, updatePolicyDisabled)
    ]);
}
