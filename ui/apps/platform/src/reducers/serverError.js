import { combineReducers } from 'redux';

export const types = {
    RECORD_SERVER_ERROR: 'serverError/RECORD_ERROR',
    RECORD_SERVER_SUCCESS: 'serverError/RECORD_SUCCESS',
};

export const actions = {
    recordServerError: () => ({ type: types.RECORD_SERVER_ERROR }),
    recordServerSuccess: () => ({ type: types.RECORD_SERVER_SUCCESS }),
};

export const serverStates = { UP: 'UP', UNREACHABLE: 'UNREACHABLE', RESURRECTED: 'RESURRECTED' };

const getStateAfterSuccessfulRequest = (state = null) => {
    // prettier-ignore
    const newServerState =
        (!state || state === serverStates.UP)
            ? serverStates.UP
            : serverStates.RESURRECTED;

    return {
        numSuccessiveFailures: 0,
        firstFailure: null,
        serverState: newServerState,
    };
};

/**
 * TODO: extract these constants to a settings file, or better yet, env. vars
 */
const MIN_FAILURES_BEFORE_FAIL = 5;
const MIN_SECONDS_BEFORE_FAIL = 15;

const checkUnreachableState = (numSuccessiveFailures, firstFailure, currentServerState) => ({
    numSuccessiveFailures,
    firstFailure,
    serverState:
        numSuccessiveFailures >= MIN_FAILURES_BEFORE_FAIL &&
        (Date.now() - firstFailure) / 1000 >= MIN_SECONDS_BEFORE_FAIL
            ? serverStates.UNREACHABLE
            : currentServerState,
});

const serverError = (state = null, action) => {
    if (action.type === types.RECORD_SERVER_SUCCESS) {
        // This is the happy path, and also an extremely hot code path since it's called with every server request.
        // We need to return the same state object where possible, because if we don't, React will do a lot of state updates
        // and slow the UI down.
        return state && state.numSuccessiveFailures === 0
            ? state
            : getStateAfterSuccessfulRequest(state && state.serverState);
    }

    if (action.type === types.RECORD_SERVER_ERROR) {
        return checkUnreachableState(
            state ? state.numSuccessiveFailures + 1 : 1,
            state && state.firstFailure ? state.firstFailure : Date.now(),
            state.serverState
        );
    }
    return state;
};

const reducer = combineReducers({
    serverError,
});

const getServerState = (state) => state.serverError && state.serverError.serverState;

export const selectors = {
    getServerState,
};

export default reducer;
