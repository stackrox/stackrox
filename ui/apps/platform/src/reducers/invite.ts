import { Reducer, combineReducers } from 'redux';

// Action types

export type SetInviteModalVisibilityAction = {
    type: 'invite/SET_MODAL_VISIBILITY';
    show: boolean;
};

// Action creator functions

export const actions = {
    setInviteModalVisibility: (show: boolean): SetInviteModalVisibilityAction => ({
        type: 'invite/SET_MODAL_VISIBILITY',
        show,
    }),
};

// Reducers

const showInviteModal: Reducer<boolean, SetInviteModalVisibilityAction> = (
    state = false,
    action
) => {
    switch (action.type) {
        case 'invite/SET_MODAL_VISIBILITY':
            return action.show;
        default:
            return state;
    }
};

const reducer = combineReducers({
    showInviteModal,
});

export default reducer;

// Selectors

type State = ReturnType<typeof reducer>;

const inviteSelector = (state: State) => state.showInviteModal;

export const selectors = {
    inviteSelector,
};
