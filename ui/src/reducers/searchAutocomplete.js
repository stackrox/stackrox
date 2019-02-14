import { combineReducers } from 'redux';

export const types = {
    SEND_AUTOCOMPLETE_REQUEST: 'autocomplete/SEND_AUTOCOMPLETE_REQUEST',
    RECORD_AUTOCOMPLETE_RESPONSE: 'autocomplete/RECORD_AUTOCOMPLETE_RESPONSE',
    CLEAR_AUTOCOMPLETE: 'autocomplete/CLEAR_AUTOCOMPLETE'
};

export const actions = {
    sendAutoCompleteRequest: request => ({
        type: types.SEND_AUTOCOMPLETE_REQUEST,
        ...request
    }),
    recordAutoCompleteResponse: autoCompleteResults => ({
        type: types.RECORD_AUTOCOMPLETE_RESPONSE,
        autoCompleteResults
    }),
    clearAutoComplete: () => ({ type: types.CLEAR_AUTOCOMPLETE })
};

const autoCompleteResults = (state = [], action) => {
    if (action.type === types.RECORD_AUTOCOMPLETE_RESPONSE) {
        return action.autoCompleteResults;
    }
    if (action.type === types.CLEAR_AUTOCOMPLETE) {
        return [];
    }
    return state;
};

const reducer = combineReducers({
    autoCompleteResults
});

const getAutoCompleteResults = state => state.autoCompleteResults;

export const selectors = {
    getAutoCompleteResults
};

export default reducer;
