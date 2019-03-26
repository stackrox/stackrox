import { combineReducers } from 'redux';
import bindSelectors from 'utils/bindSelectors';

import backendReducer, { selectors as backendSelectors } from 'reducers/network/backend';
import dialogueReducer, { selectors as dialogueSelectors } from 'reducers/network/dialogue';
import graphReducer, { selectors as graphSelectors } from 'reducers/network/graph';
import pageReducer, { selectors as pageSelectors } from 'reducers/network/page';
import searchReducer, { selectors as searchSelectors } from 'reducers/network/search';
import wizardReducer, { selectors as wizardSelectors } from 'reducers/network/wizard';

// File combines all of the reducers and selectors under reducers/network.

// Reducers
//---------

const reducer = combineReducers({
    backend: backendReducer,
    dialogue: dialogueReducer,
    graph: graphReducer,
    page: pageReducer,
    search: searchReducer,
    wizard: wizardReducer
});

export default reducer;

// Selectors
//----------

const getBackend = state => state.backend;
const getDialogue = state => state.dialogue;
const getGraph = state => state.graph;
const getPage = state => state.page;
const getSearch = state => state.search;
const getWizard = state => state.wizard;

export const selectors = {
    ...bindSelectors(getBackend, backendSelectors),
    ...bindSelectors(getDialogue, dialogueSelectors),
    ...bindSelectors(getGraph, graphSelectors),
    ...bindSelectors(getPage, pageSelectors),
    ...bindSelectors(getSearch, searchSelectors),
    ...bindSelectors(getWizard, wizardSelectors)
};
