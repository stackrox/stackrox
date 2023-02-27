import { Reducer, combineReducers } from 'redux';

// Action types

export type ServerResponseSuccessAction = {
    type: 'serverStatus/RESPONSE_SUCCESS';
};

export type ServerResponseFailureAction = {
    type: 'serverStatus/RESPONSE_FAILURE';
    now: number; // action creator must call Date.now() because reducer is pure function!
};

type ServerResponseAction = ServerResponseSuccessAction | ServerResponseFailureAction;

// Reducer

/*
 * TODO: extract these constants to a settings file, or better yet, env. vars
 */
const MIN_FAILURES_BEFORE_FAIL = 5;
const MIN_SECONDS_BEFORE_FAIL = 15;
const MIN_MILLISECONDS_BEFORE_FAIL = 1000 * MIN_SECONDS_BEFORE_FAIL;

type ServerStatus = '' | 'RESURRECTED' | 'UNREACHABLE' | 'UP';

type ServerResponseStatus = {
    firstFailure: number;
    numSuccessiveFailures: number;
    serverStatus: ServerStatus;
};

const serverResponseStatusInitialState: ServerResponseStatus = {
    firstFailure: 0,
    numSuccessiveFailures: 0,
    serverStatus: '',
};

const serverResponseStatus: Reducer<ServerResponseStatus, ServerResponseAction> = (
    state = serverResponseStatusInitialState,
    action
) => {
    switch (action.type) {
        case 'serverStatus/RESPONSE_SUCCESS': {
            // This is the happy path, and also an extremely hot code path since it's called with every server request.
            // We need to return the same state object where possible, because if we don't, React will do a lot of state updates
            // and slow the UI down.
            return state.serverStatus && state.numSuccessiveFailures === 0
                ? state
                : {
                      firstFailure: 0,
                      numSuccessiveFailures: 0,
                      serverStatus:
                          state.serverStatus === '' || state.serverStatus === 'UP'
                              ? 'UP'
                              : 'RESURRECTED',
                  };
        }

        case 'serverStatus/RESPONSE_FAILURE': {
            const firstFailure = state.firstFailure || action.now;
            const numSuccessiveFailures = state.numSuccessiveFailures + 1;
            const serverStatus =
                numSuccessiveFailures >= MIN_FAILURES_BEFORE_FAIL &&
                action.now - state.firstFailure >= MIN_MILLISECONDS_BEFORE_FAIL
                    ? 'UNREACHABLE'
                    : state.serverStatus;

            return {
                firstFailure,
                numSuccessiveFailures,
                serverStatus,
            };
        }

        default:
            return state;
    }
};

const reducer = combineReducers({
    serverResponseStatus,
});

// Selectors

type State = ReturnType<typeof reducer>;

const serverStatusSelector = (state: State) => state.serverResponseStatus.serverStatus;

export const selectors = {
    serverStatusSelector,
};

export default reducer;
