import reducer, { actions } from './serverError';

describe('Server Error Reducer', () => {
    const initialTimestamp = 946684799000; // 1999-12-31T23:59:59+0000
    const initialState = {
        serverError: null
    };
    const successState = {
        serverError: {
            numSuccessiveFailures: 0,
            firstFailure: null,
            serverState: 'UP'
        }
    };

    let realDatenow;

    beforeEach(() => {
        realDatenow = Date.now;
        Date.now = jest.fn(() => initialTimestamp);
    });

    it('should return the initial state', () => {
        expect(reducer(undefined, {})).toEqual(initialState);
    });

    it('should set the UP state when first API call succeeds', () => {
        const prevState = {
            ...initialState
        };

        const nextState = reducer(prevState, actions.recordServerSuccess());

        expect(nextState).toEqual(successState);
    });

    it('should return the same state when UP and API call succeeds', () => {
        const prevState = {
            ...successState
        };

        const nextState = reducer(prevState, actions.recordServerSuccess());

        expect(nextState).toEqual(successState);
    });

    it('should start tracking failures when an API call fails', () => {
        const firstFailureState = {
            serverError: {
                numSuccessiveFailures: 1,
                firstFailure: initialTimestamp,
                serverState: 'UP'
            }
        };

        const prevState = {
            ...successState
        };

        const nextState = reducer(prevState, actions.recordServerError());

        expect(nextState).toEqual(firstFailureState);
    });

    it('should increment the failure count when another API call fails', () => {
        const firstFailureState = {
            serverError: {
                numSuccessiveFailures: 1,
                firstFailure: initialTimestamp,
                serverState: 'UP'
            }
        };

        const nextFailureState = {
            serverError: {
                numSuccessiveFailures: 2,
                firstFailure: initialTimestamp,
                serverState: 'UP'
            }
        };
        const prevState = {
            ...firstFailureState
        };
        const nextState = reducer(prevState, actions.recordServerError());

        expect(nextState).toEqual(nextFailureState);
    });

    it('should toggle to UNREACHABLE when a fifth API call fails, at least 15 seconds after first failure', () => {
        const penultimateFailureState = {
            serverError: {
                numSuccessiveFailures: 4,
                firstFailure: initialTimestamp,
                serverState: 'UP'
            }
        };

        const nextFailureState = {
            serverError: {
                numSuccessiveFailures: 5,
                firstFailure: initialTimestamp,
                serverState: 'UNREACHABLE'
            }
        };
        const prevState = {
            ...penultimateFailureState
        };

        Date.now = jest.fn(() => initialTimestamp + 15001); // tick the "clock" ahead 15 secs.

        const nextState = reducer(prevState, actions.recordServerError());

        expect(nextState).toEqual(nextFailureState);

        Date.now = jest.fn(() => initialTimestamp); // restore first mock
    });

    it('should toggle to RESURRECTED when a fifth API call fails, at least 15 seconds after first failure', () => {
        const finalFailureState = {
            serverError: {
                numSuccessiveFailures: 5,
                firstFailure: initialTimestamp,
                serverState: 'UNREACHABLE'
            }
        };

        const nextFailureState = {
            serverError: {
                numSuccessiveFailures: 0,
                firstFailure: null,
                serverState: 'RESURRECTED'
            }
        };
        const prevState = {
            ...finalFailureState
        };

        const nextState = reducer(prevState, actions.recordServerSuccess());

        expect(nextState).toEqual(nextFailureState);
    });

    afterEach(() => {
        Date.now = realDatenow;
    });
});
