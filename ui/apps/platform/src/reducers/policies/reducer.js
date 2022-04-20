import { combineReducers } from 'redux';
import bindSelectors from 'utils/bindSelectors';

import searchReducer, { selectors as searchSelectors } from 'reducers/policies/search';
import wizardReducer, { selectors as wizardSelectors } from 'reducers/policies/wizard';

// File combines all of the reducers and selectors under reducers/policies.

// Reducers
//---------

const reducer = combineReducers({
    search: searchReducer,
    wizard: wizardReducer,
});

export default reducer;

// Selectors
//----------

const getSearch = (state) => state.search;
const getWizard = (state) => state.wizard;

export const selectors = {
    ...bindSelectors(getSearch, searchSelectors),
    ...bindSelectors(getWizard, wizardSelectors),
};
