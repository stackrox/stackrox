import { select, call } from 'redux-saga/effects';
import { expectSaga } from 'redux-saga-test-plan';
import { dynamic } from 'redux-saga-test-plan/providers';

import { selectors } from 'reducers';
import { actions, types } from 'reducers/deployments';
import { fetchDeploymentsLegacy as fetchDeployments } from 'services/DeploymentsService';

import saga from './deploymentSagas';
import createLocationChange from './sagaTestUtils';

describe('Deployment Sagas', () => {
    it('should get unfiltered list of deployments on a Dashboard and Policies pages', () => {
        const deployments = ['dep1', 'dep2'];
        const fetchMock = jest.fn().mockReturnValue({ response: deployments });

        return expectSaga(saga)
            .provide([
                [select(selectors.getDeploymentsSearchOptions), []],
                [select(selectors.getDashboardSearchOptions), []],
                [select(selectors.getPoliciesSearchOptions), []],
                [call(fetchDeployments, []), dynamic(fetchMock)],
            ])
            .dispatch(createLocationChange('/main/dashboard'))
            .dispatch({ type: types.SET_SEARCH_OPTIONS, payload: { options: [] } })
            .put(actions.fetchDeployments.success(deployments, { options: [] }))
            .dispatch(createLocationChange('/main/policies/policyId'))
            .dispatch({ type: types.SET_SEARCH_OPTIONS, payload: { options: [] } })
            .put(actions.fetchDeployments.success(deployments, { options: [] }))
            .silentRun();
    });
});
