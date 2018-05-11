import { select, call } from 'redux-saga/effects';
import { expectSaga } from 'redux-saga-test-plan';
import { dynamic } from 'redux-saga-test-plan/providers';

import { selectors } from 'reducers';
import { actions } from 'reducers/deployments';
import { types as locationActionTypes } from 'reducers/routes';
import { fetchDeployments, fetchDeployment } from 'services/DeploymentsService';
import saga from './deploymentSagas';

const deploymentTypeSearchOptions = [
    {
        value: 'Deployment Type:',
        label: 'Deployment Type:',
        type: 'categoryOption'
    },
    {
        value: 'bla',
        label: 'bla'
    }
];

const createLocationChange = pathname => ({
    type: locationActionTypes.LOCATION_CHANGE,
    payload: { pathname }
});

describe('Deployment Sagas', () => {
    it('should get unfiltered list of deployments on a Dashboard and Policies pages', () => {
        const deployments = ['dep1', 'dep2'];
        const fetchMock = jest.fn().mockReturnValue({ response: deployments });

        return expectSaga(saga)
            .provide([
                [select(selectors.getDeploymentsSearchOptions), deploymentTypeSearchOptions],
                [call(fetchDeployments, []), dynamic(fetchMock)]
            ])
            .put(actions.fetchDeployments.success(deployments, { options: [] }))
            .dispatch(createLocationChange('/main/dashboard'))
            .put(actions.fetchDeployments.success(deployments, { options: [] }))
            .dispatch(createLocationChange('/main/policies/policyId'))
            .silentRun();
    });

    it('should get a filtered list of deployments on a Risk page', () => {
        const deployments = ['dep1', 'dep2'];
        const fetchMock = jest.fn().mockReturnValueOnce({ response: deployments });

        return expectSaga(saga)
            .provide([
                [select(selectors.getDeploymentsSearchOptions), deploymentTypeSearchOptions],
                [call(fetchDeployments, deploymentTypeSearchOptions), dynamic(fetchMock)]
            ])
            .put(
                actions.fetchDeployments.success(deployments, {
                    options: deploymentTypeSearchOptions
                })
            )
            .dispatch(createLocationChange('/main/risk'))
            .silentRun();
    });

    it('should re-fetch deployments with new search options', () => {
        const deployments = ['dep1', 'dep2'];
        const fetchMock = jest.fn().mockReturnValueOnce({ response: deployments });

        return expectSaga(saga)
            .provide([[call(fetchDeployments, deploymentTypeSearchOptions), dynamic(fetchMock)]])
            .put(
                actions.fetchDeployments.success(deployments, {
                    options: deploymentTypeSearchOptions
                })
            )
            .dispatch(actions.setDeploymentsSearchOptions(deploymentTypeSearchOptions))
            .silentRun();
    });

    it('should fetch deployment details on a Risk page with deployment selected', () => {
        const deployment = { id: 'dep' };
        const fetchMock = jest.fn().mockReturnValueOnce({ response: deployment });

        return expectSaga(saga)
            .provide([
                [select(selectors.getDeploymentsSearchOptions), []],
                [call(fetchDeployment, deployment.id), dynamic(fetchMock)]
            ])
            .put(actions.fetchDeployment.success(deployment, { id: deployment.id }))
            .dispatch(createLocationChange(`/main/risk/${deployment.id}`))
            .silentRun();
    });
});
