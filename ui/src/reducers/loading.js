import { combineReducers } from 'redux';
import { filterRequestActionTypes, getFetchingActionName } from 'utils/fetchingReduxRoutines';

const loading = (state = {}, action) => {
    const { type } = action;
    const matches = filterRequestActionTypes(type);

    // not a *_REQUEST / *_SUCCESS /  *_FAILURE actions, so we ignore them
    if (!matches) return state;

    const { requestName, requestState } = matches;
    return {
        ...state,
        // Store whether a request is happening at the moment or not
        // e.g. will be true when receiving GET_TODOS_REQUEST
        //      and false when receiving GET_TODOS_SUCCESS / GET_TODOS_FAILURE
        [requestName]: requestState
    };
};

const reducer = combineReducers({
    loading
});

export default reducer;

const getLoadingStatus = (state, type) => {
    const action = getFetchingActionName(type);
    return state.loading[action];
};

export const selectors = {
    getLoadingStatus
};
