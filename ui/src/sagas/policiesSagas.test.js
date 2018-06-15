import { select, call } from 'redux-saga/effects';
import { expectSaga } from 'redux-saga-test-plan';
import { dynamic } from 'redux-saga-test-plan/providers';
import { push } from 'react-router-redux';

import { selectors } from 'reducers';
import { actions, types } from 'reducers/policies';
import { actions as notificationActions } from 'reducers/notifications';
import { types as locationActionTypes } from 'reducers/routes';
import * as service from 'services/PoliciesService';
import saga from './policiesSagas';

const policyTypeSearchOptions = [
    {
        value: 'name:',
        label: 'name:',
        type: 'categoryOption'
    },
    {
        value: 'test',
        label: 'test'
    }
];

const policiesSearchQuery = {
    query: 'name:test'
};

const policy = {
    name: 'test123',
    id: '12345',
    severity: 'HIGH_SEVERITY',
    categories: ['Container Configuration'],
    disabled: false,
    imagePolicy: { imageName: { namespace: 'test' } }
};

const policies = ['pol1', 'pol2'];

const policyCategories = policies;

const dryRun = {
    alerts: [],
    excluded: []
};

const createLocationChange = pathname => ({
    type: locationActionTypes.LOCATION_CHANGE,
    payload: { pathname }
});

describe('Policies Sagas', () => {
    it('should get unfiltered list of policies and policy categories on Policies page', () => {
        const fetchPoliciesMock = jest.fn().mockReturnValue({ response: policies });
        const fetchPolicyCategoriesMock = jest.fn().mockReturnValue({ response: policyCategories });

        return expectSaga(saga)
            .provide([
                [select(selectors.getPoliciesSearchOptions), []],
                [call(service.fetchPolicies, { query: '' }), dynamic(fetchPoliciesMock)],
                [call(service.fetchPolicyCategories), dynamic(fetchPolicyCategoriesMock)]
            ])
            .put(actions.fetchPolicies.success(policies))
            .put(actions.fetchPolicyCategories.success(policyCategories))
            .dispatch({ type: types.SET_SEARCH_OPTIONS, payload: { options: [] } })
            .dispatch(createLocationChange('/main/policies'))
            .silentRun();
    });

    it('should get filtered list of policies on Policies page', () => {
        const fetchMock = jest.fn().mockReturnValue({ response: policies });

        return expectSaga(saga)
            .provide([
                [select(selectors.getPoliciesSearchOptions), policyTypeSearchOptions],
                [call(service.fetchPolicies, policiesSearchQuery), dynamic(fetchMock)]
            ])
            .put(actions.fetchPolicies.success(policies))
            .dispatch({
                type: types.SET_SEARCH_OPTIONS,
                payload: { options: policyTypeSearchOptions }
            })
            .dispatch(createLocationChange('/main/policies'))
            .silentRun();
    });

    it('should re-fetch policies with new policies search options', () => {
        const fetchMock = jest.fn().mockReturnValueOnce({ response: policies });

        return expectSaga(saga)
            .provide([
                [select(selectors.getPoliciesSearchOptions), policyTypeSearchOptions],
                [call(service.fetchPolicies, policiesSearchQuery), dynamic(fetchMock)]
            ])
            .put(actions.fetchPolicies.success(policies))
            .dispatch({
                type: types.SET_SEARCH_OPTIONS,
                payload: { options: policyTypeSearchOptions }
            })
            .dispatch(actions.setPoliciesSearchOptions(policyTypeSearchOptions))
            .silentRun();
    });

    it('should fetch policy details on Policies page with policy selected', () => {
        const policyId = '12345';
        const fetchMock = jest.fn().mockReturnValueOnce({ response: policy });

        return expectSaga(saga)
            .provide([
                [select(selectors.getPoliciesSearchOptions), []],
                [call(service.fetchPolicy, policyId), dynamic(fetchMock)]
            ])
            .put(actions.fetchPolicy.success(policy))
            .dispatch(createLocationChange(`/main/policies/${policyId}`))
            .silentRun();
    });

    it('should reassess policies and show toast', () =>
        expectSaga(saga)
            .provide([[call(service.reassessPolicies), {}]])
            .put(notificationActions.addNotification('Policies were reassessed'))
            .put(notificationActions.removeOldestNotification())
            .dispatch({ type: types.REASSESS_POLICIES })
            .silentRun());

    it('should delete policies', () =>
        expectSaga(saga)
            .provide([
                [select(selectors.getPoliciesSearchOptions), []],
                [call(service.deletePolicies), ['12345', '6789']]
            ])
            .dispatch({ type: types.DELETE_POLICIES })
            .silentRun());

    it('should update given policy', () => {
        const saveMock = jest.fn().mockReturnValueOnce({});

        return expectSaga(saga)
            .provide([
                [select(selectors.getPoliciesSearchOptions), []],
                [call(service.savePolicy, policy), dynamic(saveMock)]
            ])
            .dispatch({ type: types.UPDATE_POLICY, policy })
            .silentRun()
            .then(() => {
                expect(saveMock.mock.calls.length).toBe(1);
            });
    });

    it('should create new policy with specified policy payload on CREATE wizard state', () => {
        const createMock = jest.fn().mockReturnValueOnce({ data: policy });

        return expectSaga(saga)
            .provide([
                [select(selectors.getPoliciesSearchOptions), []],
                [call(service.createPolicy, policy), dynamic(createMock)]
            ])
            .put(actions.setPolicyWizardState({ current: '', isNew: false }))
            .put(push(`/main/policies/${policy.id}`))
            .dispatch({
                type: types.SET_POLICY_WIZARD_STATE,
                state: { current: 'CREATE', policy }
            })
            .silentRun()
            .then(() => {
                expect(createMock.mock.calls.length).toBe(1);
            });
    });

    it('should save existing policy with specified policy payload on SAVE wizard state', () => {
        const saveMock = jest.fn().mockReturnValueOnce({ data: policy });

        return expectSaga(saga)
            .provide([
                [select(selectors.getPoliciesSearchOptions), []],
                [call(service.savePolicy, policy), dynamic(saveMock)]
            ])
            .put(actions.setPolicyWizardState({ current: '', isNew: false }))
            .dispatch({
                type: types.SET_POLICY_WIZARD_STATE,
                state: { current: 'SAVE', policy }
            })
            .silentRun()
            .then(() => {
                expect(saveMock.mock.calls.length).toBe(1);
            });
    });

    it('should get policy dry run on PRE_PREVIEW wizard state', () => {
        const dryRunMock = jest.fn().mockReturnValueOnce({ data: dryRun });

        return expectSaga(saga)
            .provide([[call(service.getDryRun, policy), dynamic(dryRunMock)]])
            .put(
                actions.setPolicyWizardState({
                    current: 'PREVIEW',
                    dryrun: dryRun,
                    policy
                })
            )
            .dispatch({
                type: types.SET_POLICY_WIZARD_STATE,
                state: { current: 'PRE_PREVIEW', policy }
            })
            .silentRun();
    });
});
