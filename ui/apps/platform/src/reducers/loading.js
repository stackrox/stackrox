import { combineReducers } from 'redux';
import { FetchingActionState, getFetchingActionInfo } from 'utils/fetchingReduxRoutines';

const loading = (state = {}, action) => {
    const { type } = action;
    const info = getFetchingActionInfo(type);

    // not a *_REQUEST / *_SUCCESS /  *_FAILURE actions, so we ignore them
    if (!info) {
        return state;
    }

    const { prefix, fetchingState } = info;
    return {
        ...state,
        // Store whether a request is happening at the moment or not
        // e.g. will be true when receiving GET_TODOS_REQUEST
        //      and false when receiving GET_TODOS_SUCCESS / GET_TODOS_FAILURE
        [prefix]: fetchingState === FetchingActionState.REQUEST,
    };
};

const reducer = combineReducers({
    loading,
});

export default reducer;

const getLoadingStatus = (state, fetchingActionTypes) => {
    const info = getFetchingActionInfo(fetchingActionTypes.REQUEST);
    if (!info) {
        return false;
    }
    return state.loading[info.prefix] || false;
};

export const selectors = {
    getLoadingStatus,
};
