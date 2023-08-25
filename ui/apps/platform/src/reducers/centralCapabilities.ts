import { Reducer, combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { PrefixedAction } from 'utils/fetchingReduxRoutines';
import { fetchCentralCapabilities, CentralServicesCapabilities } from 'services/MetadataService';

// Types

export type CentralCapabilitiesAction = PrefixedAction<
    'metadata/FETCH_CENTRAL_CAPABILITIES',
    CentralServicesCapabilities
>;

// Actions

export const fetchCentralCapabilitiesThunk = () => {
    return async (dispatch) => {
        dispatch({ type: 'metadata/FETCH_CENTRAL_CAPABILITIES_REQUEST' });

        try {
            const data = await fetchCentralCapabilities();
            dispatch({
                type: 'metadata/FETCH_CENTRAL_CAPABILITIES_SUCCESS',
                response: data,
            });
        } catch (error) {
            dispatch({ type: 'metadata/FETCH_CENTRAL_CAPABILITIES_FAILURE', error });
        }
    };
};

// Reducers

const centralCapabilities: Reducer<CentralServicesCapabilities, CentralCapabilitiesAction> = (
    state = {} as CentralServicesCapabilities,
    action
) => {
    if (action.type === 'metadata/FETCH_CENTRAL_CAPABILITIES_SUCCESS') {
        return isEqual(action.response, state) ? state : action.response;
    }
    return state;
};

const centralCapabilitiesError: Reducer<Error | null, CentralCapabilitiesAction> = (
    state = null,
    action
) => {
    switch (action.type) {
        case 'metadata/FETCH_CENTRAL_CAPABILITIES_REQUEST':
        case 'metadata/FETCH_CENTRAL_CAPABILITIES_SUCCESS':
            return null;

        case 'metadata/FETCH_CENTRAL_CAPABILITIES_FAILURE':
            return action.error;

        default:
            return state;
    }
};

const isLoadingCentralCapabilities = (state = true, action) => {
    switch (action.type) {
        case 'metadata/FETCH_CENTRAL_CAPABILITIES_REQUEST':
            return true;

        case 'metadata/FETCH_CENTRAL_CAPABILITIES_FAILURE':
        case 'metadata/FETCH_CENTRAL_CAPABILITIES_SUCCESS':
            return false;

        default:
            return state;
    }
};

const reducer = combineReducers({
    centralCapabilities,
    centralCapabilitiesError,
    isLoadingCentralCapabilities,
});

type State = ReturnType<typeof reducer>;

const getCentralCapabilities = (state: State) => state.centralCapabilities;
const getCentralCapabilitiesError = (state: State) => state.centralCapabilitiesError;
const getIsLoadingCentralCapabilities = (state: State) => state.isLoadingCentralCapabilities;

export const selectors = {
    getCentralCapabilities,
    getCentralCapabilitiesError,
    getIsLoadingCentralCapabilities,
};

export default reducer;
