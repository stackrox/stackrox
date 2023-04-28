import { Reducer, combineReducers } from 'redux';

// Action types

export type SetFeedbackModalVisibilityAction = {
    type: 'feedback/SET_MODAL_VISIBILITY';
    show: boolean;
};

// Action creator functions

export const actions = {
    setFeedbackModalVisibility: (show: boolean): SetFeedbackModalVisibilityAction => ({
        type: 'feedback/SET_MODAL_VISIBILITY',
        show,
    }),
};

// Reducers

const showFeedbackModal: Reducer<boolean, SetFeedbackModalVisibilityAction> = (
    state = false,
    action
) => {
    switch (action.type) {
        case 'feedback/SET_MODAL_VISIBILITY':
            return action.show;
        default:
            return state;
    }
};

const reducer = combineReducers({
    showFeedbackModal,
});

export default reducer;

// Selectors

type State = ReturnType<typeof reducer>;

const feedbackSelector = (state: State) => state.showFeedbackModal;

export const selectors = {
    feedbackSelector,
};
