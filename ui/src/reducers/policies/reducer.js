import { combineReducers } from 'redux';
import bindSelectors from 'utils/bindSelectors';

import backendReducer, { selectors as backendSelectors } from 'reducers/policies/backend';
import pageReducer, { selectors as pageSelectors } from 'reducers/policies/page';
import searchReducer, { selectors as searchSelectors } from 'reducers/policies/search';
import tableReducer, { selectors as tableSelectors } from 'reducers/policies/table';
import wizardReducer, { selectors as wizardSelectors } from 'reducers/policies/wizard';

// File combines all of the reducers and selectors under reducers/policies.

// Reducers
//---------

const reducer = combineReducers({
    backend: backendReducer,
    page: pageReducer,
    search: searchReducer,
    table: tableReducer,
    wizard: wizardReducer
});

export default reducer;

// Selectors
//----------

const getBackend = state => state.backend;
const getPage = state => state.page;
const getSearch = state => state.search;
const getTable = state => state.table;
const getWizard = state => state.wizard;

export const selectors = {
    ...bindSelectors(getBackend, backendSelectors),
    ...bindSelectors(getPage, pageSelectors),
    ...bindSelectors(getSearch, searchSelectors),
    ...bindSelectors(getTable, tableSelectors),
    ...bindSelectors(getWizard, wizardSelectors)
};
