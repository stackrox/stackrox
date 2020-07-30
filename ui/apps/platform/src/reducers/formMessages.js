import { combineReducers } from 'redux';

// Action types

export const types = {
    ADD_FORM_MESSAGE: 'formMessages/ADD_FORM_MESSAGE',
    CLEAR_FORM_MESSAGES: 'formMessages/CLEAR_FORM_MESSAGES',
};

// Actions

export const actions = {
    addFormMessage: (formMessage) => ({
        type: types.ADD_FORM_MESSAGE,
        formMessage,
    }),
    clearFormMessages: () => ({ type: types.CLEAR_FORM_MESSAGES }),
};

// Reducers

const formMessages = (state = [], action) => {
    const newState = state.slice();
    switch (action.type) {
        case types.ADD_FORM_MESSAGE:
            newState.push(action.formMessage);
            return newState;
        case types.CLEAR_FORM_MESSAGES:
            return [];
        default:
            return state;
    }
};

const reducer = combineReducers({
    formMessages,
});

export default reducer;

// Selectors

const getFormMessages = (state) => state.formMessages;

export const selectors = {
    getFormMessages,
};
