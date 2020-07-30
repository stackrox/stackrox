import reducer, { actions } from './formMessages';

const initialState = {
    formMessages: [],
};

const formMessage = 'formMessage added';

describe('FormMessages Reducer', () => {
    it('should return the initial state', () => {
        expect(reducer(undefined, {})).toEqual(initialState);
    });

    it('should add new formMessages when receiving a new formMessage', () => {
        const prevState = initialState;
        const nextState = reducer(prevState, actions.addFormMessage(formMessage));
        expect(nextState.formMessages).toEqual([formMessage]);
    });

    it('should clear all formMessage', () => {
        const prevState = {
            formMessages: [formMessage, 'second message'],
        };
        const nextState = reducer(prevState, actions.clearFormMessages());
        expect(nextState.formMessages).toEqual([]);
    });
});
