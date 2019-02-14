import { combineReducers } from 'redux';
import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

// Action types

export const types = {
    FETCH_PDF: createFetchingActionTypes('pdf/FETCH_PDF')
};

// Actions

export const actions = {
    fetchPdf: createFetchingActions(types.FETCH_PDF)
};

// Reducers

const pdfLoadingStatus = (state = false, action) => {
    if (action.type === types.FETCH_PDF.REQUEST || action.type === types.FETCH_PDF.SUCCESS) {
        return !state;
    }
    return state;
};
const reducer = combineReducers({
    pdfLoadingStatus
});

export default reducer;

// Selectors

const getPdfLoadingStatus = state => state.pdfLoadingStatus;

export const selectors = {
    getPdfLoadingStatus
};
