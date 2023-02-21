import reducer from './serverResponseStatus';

describe('Server Error Reducer', () => {
    const initialTimestamp = 946684799000; // 1999-12-31T23:59:59+0000
    const initialState = {
        serverResponseStatus: {
            firstFailure: 0,
            numSuccessiveFailures: 0,
            serverStatus: '',
        },
    };
    const successState = {
        serverResponseStatus: {
            firstFailure: 0,
            numSuccessiveFailures: 0,
            serverStatus: 'UP',
        },
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
            ...initialState,
        };

        const nextState = reducer(prevState, { type: 'serverStatus/RESPONSE_SUCCESS' });

        expect(nextState).toEqual(successState);
    });

    it('should return the same state when UP and API call succeeds', () => {
        const prevState = {
            ...successState,
        };

        const nextState = reducer(prevState, { type: 'serverStatus/RESPONSE_SUCCESS' });

        expect(nextState).toEqual(successState);
    });

    it('should start tracking failures when an API call fails', () => {
        const firstFailureState = {
            serverResponseStatus: {
                numSuccessiveFailures: 1,
                firstFailure: initialTimestamp,
                serverStatus: 'UP',
            },
        };

        const prevState = {
            ...successState,
        };

        const nextState = reducer(prevState, {
            type: 'serverStatus/RESPONSE_FAILURE',
            now: Date.now(),
        });

        expect(nextState).toEqual(firstFailureState);
    });

    it('should increment the failure count when another API call fails', () => {
        const firstFailureState = {
            serverResponseStatus: {
                numSuccessiveFailures: 1,
                firstFailure: initialTimestamp,
                serverStatus: 'UP',
            },
        };

        const nextFailureState = {
            serverResponseStatus: {
                numSuccessiveFailures: 2,
                firstFailure: initialTimestamp,
                serverStatus: 'UP',
            },
        };
        const prevState = {
            ...firstFailureState,
        };
        const nextState = reducer(prevState, {
            type: 'serverStatus/RESPONSE_FAILURE',
            now: Date.now(),
        });

        expect(nextState).toEqual(nextFailureState);
    });

    it('should toggle to UNREACHABLE when a fifth API call fails, at least 15 seconds after first failure', () => {
        const penultimateFailureState = {
            serverResponseStatus: {
                numSuccessiveFailures: 4,
                firstFailure: initialTimestamp,
                serverStatus: 'UP',
            },
        };

        const nextFailureState = {
            serverResponseStatus: {
                numSuccessiveFailures: 5,
                firstFailure: initialTimestamp,
                serverStatus: 'UNREACHABLE',
            },
        };
        const prevState = {
            ...penultimateFailureState,
        };

        Date.now = jest.fn(() => initialTimestamp + 15001); // tick the "clock" ahead 15 secs.

        const nextState = reducer(prevState, {
            type: 'serverStatus/RESPONSE_FAILURE',
            now: Date.now(),
        });

        expect(nextState).toEqual(nextFailureState);

        Date.now = jest.fn(() => initialTimestamp); // restore first mock
    });

    it('should toggle to RESURRECTED when a fifth API call fails, at least 15 seconds after first failure', () => {
        const finalFailureState = {
            serverResponseStatus: {
                numSuccessiveFailures: 5,
                firstFailure: initialTimestamp,
                serverStatus: 'UNREACHABLE',
            },
        };

        const nextFailureState = {
            serverResponseStatus: {
                numSuccessiveFailures: 0,
                firstFailure: 0,
                serverStatus: 'RESURRECTED',
            },
        };
        const prevState = {
            ...finalFailureState,
        };

        const nextState = reducer(prevState, { type: 'serverStatus/RESPONSE_SUCCESS' });

        expect(nextState).toEqual(nextFailureState);
    });

    afterEach(() => {
        Date.now = realDatenow;
    });
});
