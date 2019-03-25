import { select, call } from 'redux-saga/effects';
import { expectSaga } from 'redux-saga-test-plan';
import { dynamic } from 'redux-saga-test-plan/providers';
import { push } from 'react-router-redux';

import { selectors } from 'reducers';
import { actions as backendActions, types as backendTypes } from 'reducers/policies/backend';
import { actions as pageActions } from 'reducers/policies/page';
import { actions as searchActions, types as searchTypes } from 'reducers/policies/search';
import { actions as tableActions } from 'reducers/policies/table';
import { actions as wizardActions } from 'reducers/policies/wizard';
import { actions as notificationActions } from 'reducers/notifications';
import { types as locationActionTypes } from 'reducers/routes';
import wizardStages from 'Containers/Policies/Wizard/wizardStages';
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
            .put(backendActions.fetchPolicies.success(policies))
            .put(backendActions.fetchPolicyCategories.success(policyCategories))
            .dispatch({ type: searchTypes.SET_SEARCH_OPTIONS, payload: { options: [] } })
            .dispatch(createLocationChange('/main/policies'))
            .silentRun();
    });

    it('should get filtered list of policies on Policies page', () => {
        const fetchMock = jest.fn().mockReturnValue({ response: policies });

        return expectSaga(saga)
            .provide([
                [select(selectors.getPoliciesSearchOptions), policyTypeSearchOptions],
                [call(service.fetchPolicies, policiesSearchQuery), dynamic(fetchMock)],
                [call(service.fetchPolicyCategories), { response: policyCategories }]
            ])
            .put(backendActions.fetchPolicies.success(policies))
            .dispatch({
                type: searchTypes.SET_SEARCH_OPTIONS,
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
            .put(backendActions.fetchPolicies.success(policies))
            .dispatch({
                type: searchTypes.SET_SEARCH_OPTIONS,
                payload: { options: policyTypeSearchOptions }
            })
            .dispatch(searchActions.setPoliciesSearchOptions(policyTypeSearchOptions))
            .silentRun();
    });

    it('should fetch policy details on Policies page with policy selected', () => {
        const policyId = '12345';
        const response = { entities: { policy: [policy] } };
        const fetchMock = jest.fn().mockReturnValueOnce({ response });

        return expectSaga(saga)
            .provide([
                [select(selectors.getPoliciesSearchOptions), []],
                [call(service.fetchPolicy, policyId), dynamic(fetchMock)],
                [select(selectors.getWizardPolicy), policy],
                [call(service.fetchPolicyCategories), { response: policyCategories }],
                [call(service.fetchPolicies, { query: '' }), {}]
            ])
            .put(backendActions.fetchPolicy.success(response))
            .put(tableActions.selectPolicyId(policy.id))
            .put(wizardActions.setWizardPolicy(policy))
            .put(wizardActions.setWizardStage(wizardStages.details))
            .put(pageActions.openWizard())
            .dispatch(createLocationChange(`/main/policies/${policyId}`))
            .silentRun();
    });

    it('should reassess policies and show toast', () =>
        expectSaga(saga)
            .provide([[call(service.reassessPolicies), {}]])
            .put(notificationActions.addNotification('Policies were reassessed'))
            .put(notificationActions.removeOldestNotification())
            .dispatch({ type: backendTypes.REASSESS_POLICIES })
            .silentRun());

    it('should delete policies', () =>
        expectSaga(saga)
            .provide([
                [select(selectors.getPoliciesSearchOptions), []],
                [call(service.deletePolicies), ['12345', '6789']]
            ])
            .dispatch({ type: backendTypes.DELETE_POLICIES })
            .silentRun());

    it('should update given policy', () => {
        const saveMock = jest.fn().mockReturnValueOnce({});

        return expectSaga(saga)
            .provide([
                [select(selectors.getPoliciesSearchOptions), []],
                [call(service.savePolicy, policy), dynamic(saveMock)],
                [call(service.fetchPolicies, { query: '' }), {}]
            ])
            .dispatch({ type: backendTypes.UPDATE_POLICY, policy })
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
                [select(selectors.getWizardPolicy), policy],
                [call(service.createPolicy, policy), dynamic(createMock)],
                [call(service.fetchPolicies, { query: '' }), {}]
            ])
            .put(wizardActions.setWizardStage(wizardStages.details))
            .put(push(`/main/policies/${policy.id}`))
            .dispatch(wizardActions.setWizardStage(wizardStages.create))
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
                [select(selectors.getWizardPolicy), policy],
                [call(service.savePolicy, policy), dynamic(saveMock)],
                [call(service.fetchPolicy, policy.id), {}],
                [call(service.fetchPolicies, { query: '' }), {}],
                [call(service.fetchPolicyCategories), { response: policyCategories }]
            ])
            .put(wizardActions.setWizardStage(wizardStages.details))
            .dispatch(wizardActions.setWizardStage(wizardStages.save))
            .silentRun()
            .then(() => {
                expect(saveMock.mock.calls.length).toBe(1);
            });
    });

    it('should get policy dry run on PRE_PREVIEW wizard state', () => {
        const dryRunMock = jest.fn().mockReturnValueOnce({ data: dryRun });

        return expectSaga(saga)
            .provide([
                [call(service.getDryRun, policy), dynamic(dryRunMock)],
                [select(selectors.getWizardPolicy), policy]
            ])
            .put(wizardActions.setWizardDryRun(dryRun))
            .put(wizardActions.setWizardStage(wizardStages.preview))
            .dispatch(wizardActions.setWizardStage(wizardStages.prepreview))
            .silentRun();
    });
});
