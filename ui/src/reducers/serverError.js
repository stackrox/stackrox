import { combineReducers } from 'redux';

export const types = {
    RECORD_SERVER_ERROR: 'serverError/RECORD_ERROR',
    RECORD_SERVER_SUCCESS: 'serverError/RECORD_SUCCESS'
};

export const actions = {
    recordServerError: () => ({ type: types.RECORD_SERVER_ERROR }),
    recordServerSuccess: () => ({ type: types.RECORD_SERVER_SUCCESS })
};

const stateAfterReset = () => ({
    numSuccessiveFailures: 0,
    firstFailure: null,
    serverIsUnreachable: false
});

const MIN_FAILURES_BEFORE_FAIL = 5;
const MIN_SECONDS_BEFORE_FAIL = 15;

const deriveState = (numSuccessiveFailures, firstFailure) => ({
    numSuccessiveFailures,
    firstFailure,
    serverIsUnreachable:
        numSuccessiveFailures >= MIN_FAILURES_BEFORE_FAIL &&
        (Date.now() - firstFailure) / 1000 >= MIN_SECONDS_BEFORE_FAIL
});

const serverError = (state = null, action) => {
    if (action.type === types.RECORD_SERVER_SUCCESS) {
        // This is the happy path, and also an extremely hot code path since it's called with every server request.
        // We need to return the same state object where possible, because if we don't, React will do a lot of state updates
        // and slow the UI down.
        return state && state.numSuccessiveFailures === 0 ? state : stateAfterReset();
    }

    if (action.type === types.RECORD_SERVER_ERROR) {
        return deriveState(
            state ? state.numSuccessiveFailures + 1 : 1,
            state && state.firstFailure ? state.firstFailure : Date.now()
        );
    }
    return state;
};

const reducer = combineReducers({
    serverError
});

const getServerIsUnreachable = state => state.serverError && state.serverError.serverIsUnreachable;

export const selectors = {
    getServerIsUnreachable
};

export default reducer;
