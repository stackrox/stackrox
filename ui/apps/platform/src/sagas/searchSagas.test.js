/* eslint jest/expect-expect: ["error", { "assertFunctionNames": ["expectSaga", "expectSagaWithGlobalSearchMocked"] }] */

import { expectSaga } from 'redux-saga-test-plan';
import * as matchers from 'redux-saga-test-plan/matchers';

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
            .dispatch(createLocationChange('/main/policy-management/policies'))
            .silentRun();
    });
});
