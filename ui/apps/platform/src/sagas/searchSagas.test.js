/* eslint jest/expect-expect: ["error", { "assertFunctionNames": ["expectSaga", "expectSagaWithGlobalSearchMocked"] }] */

import { call } from 'redux-saga/effects';
import { expectSaga } from 'redux-saga-test-plan';
import * as matchers from 'redux-saga-test-plan/matchers';

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

    const expectSagaWithGlobalSearchMocked = (toProvide) =>
        expectSaga(saga).provide([[call(fetchOptions, ''), {}], ...toProvide]);

    // TODO Can this be deleted too after completion of ROX-9450 ?
    it('Should load search modifiers/suggestions for Policies when location changes to Policies Page', () => {
        const options = ['option1', 'option2'];
        return expectSagaWithGlobalSearchMocked([
            [call(fetchOptions, 'categories=POLICIES'), { options }],
        ])
            .put(policiesActions.setPoliciesSearchModifiers(options))
            .put(policiesActions.setPoliciesSearchSuggestions(options))
            .dispatch(createLocationChange('/main/policies'))
            .silentRun();
    });
});
