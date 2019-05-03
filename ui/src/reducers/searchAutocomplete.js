import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

export const types = {
    SEND_AUTOCOMPLETE_REQUEST: 'autocomplete/SEND_AUTOCOMPLETE_REQUEST',
    RECORD_AUTOCOMPLETE_RESPONSE: 'autocomplete/RECORD_AUTOCOMPLETE_RESPONSE',
    CLEAR_AUTOCOMPLETE: 'autocomplete/CLEAR_AUTOCOMPLETE',
    SET_ALL_SEARCH_OPTIONS: 'search/SET_ALL_SEARCH_OPTIONS'
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
    clearAutoComplete: () => ({ type: types.CLEAR_AUTOCOMPLETE }),
    setAllSearchOptions: options => ({ type: types.SET_ALL_SEARCH_OPTIONS, options })
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

const allSearchOptions = (state = [], action) => {
    if (action.type === 'search/SET_ALL_SEARCH_OPTIONS') {
        const { options } = action;
        if (options.length % 2 === 0) {
            return isEqual(options, state) ? state : options;
        }
    }
    return state;
};

const reducer = combineReducers({
    autoCompleteResults,
    allSearchOptions
});

const getAutoCompleteResults = state => state.autoCompleteResults;
const getAllSearchOptions = state => state.allSearchOptions;

export const selectors = {
    getAutoCompleteResults,
    getAllSearchOptions
};

export default reducer;
