import { call } from 'redux-saga/effects';
import { expectSaga } from 'redux-saga-test-plan';
import * as matchers from 'redux-saga-test-plan/matchers';

import { actions as alertActions } from 'reducers/alerts';
import { actions as deploymentsActions } from 'reducers/deployments';
import { actions as policiesActions } from 'reducers/policies/search';
import { actions as globalSearchActions } from 'reducers/globalSearch';
import { fetchOptions } from 'services/SearchService';
import createLocationChange from './sagaTestUtils';
import saga from './searchSagas';

describe('Search Sagas Test', () => {
    it('Should load search modifiers/suggestions for Global Search once when hit any `main` location', () => {
        const options = ['option1', 'option2'];
        return expectSaga(saga)
            .provide([[matchers.call.fn(fetchOptions), { options }]])
            .put(globalSearchActions.setGlobalSearchModifiers(options))
            .put(globalSearchActions.setGlobalSearchSuggestions(options))
            .not.put(globalSearchActions.setGlobalSearchModifiers(options)) // should not do it second time
            .dispatch(createLocationChange('/main/dashboard'))
            .dispatch(createLocationChange('/main/policies'))
            .silentRun();
    });

    it('Should load search modifiers/suggestions for Alerts when location changes to Violations Page', () => {
        const options = ['option1', 'option2'];
        return expectSaga(saga)
            .provide([[call(fetchOptions, 'categories=ALERTS'), { options }]])
            .put(alertActions.setAlertsSearchModifiers(options))
            .put(alertActions.setAlertsSearchSuggestions(options))
            .dispatch(createLocationChange('/main/violations'))
            .silentRun();
    });

    it('Should load search modifiers/suggestions for Deployments when location changes to Risk Page', () => {
        const options = ['option1', 'option2'];
        return expectSaga(saga)
            .provide([[call(fetchOptions, 'categories=DEPLOYMENTS'), { options }]])
            .put(deploymentsActions.setDeploymentsSearchModifiers(options))
            .put(deploymentsActions.setDeploymentsSearchSuggestions(options))
            .dispatch(createLocationChange('/main/risk'))
            .silentRun();
    });

    it('Should load search modifiers/suggestions for Policies when location changes to Policies Page', () => {
        const options = ['option1', 'option2'];
        return expectSaga(saga)
            .provide([[call(fetchOptions, 'categories=POLICIES'), { options }]])
            .put(policiesActions.setPoliciesSearchModifiers(options))
            .put(policiesActions.setPoliciesSearchSuggestions(options))
            .dispatch(createLocationChange('/main/policies'))
            .silentRun();
    });
});
