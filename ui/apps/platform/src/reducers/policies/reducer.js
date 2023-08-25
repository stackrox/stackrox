import { combineReducers } from 'redux';
import bindSelectors from 'utils/bindSelectors';

import wizardReducer, { selectors as wizardSelectors } from 'reducers/policies/wizard';

// File combines all of the reducers and selectors under reducers/policies.

// Reducers
//---------

const reducer = combineReducers({
    wizard: wizardReducer,
});

export default reducer;

// Selectors
//----------

const getWizard = (state) => state.wizard;

export const selectors = {
    ...bindSelectors(getWizard, wizardSelectors),
};
