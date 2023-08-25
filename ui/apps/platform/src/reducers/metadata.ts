import { Reducer, combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { Metadata } from 'types/metadataService.proto';
import { PrefixedAction } from 'utils/fetchingReduxRoutines';

// Action types

type InitialFetchMetadataAction = PrefixedAction<'metadata/INITIAL_FETCH_METADATA', Metadata>;
type PollMetadataAction = PrefixedAction<'metadata/POLL_METADATA', Metadata>;

type MetadataAction = InitialFetchMetadataAction | PollMetadataAction;

// Action creator functions

// TODO Replaced abstract with concrete for compatibility with sagas, but simplify to action objects for thunks, if possible.
export const actions = {
    initialFetchMetadata: {
        failure: (error: Error) => ({
            type: 'metadata/INITIAL_FETCH_METADATA_FAILURE',
            error,
        }),
        success: (response: Metadata) => ({
            type: 'metadata/INITIAL_FETCH_METADATA_SUCCESS',
            response,
        }),
    },
    pollMetadata: {
        failure: (error: Error) => ({
            type: 'metadata/POLL_METADATA_FAILURE',
            error,
        }),
        success: (response: Metadata) => ({
            type: 'metadata/POLL_METADATA_SUCCESS',
            response,
        }),
    },
};

// Reducers

// Initial state arbitrarily assumes release build.
const metadataInitialState: Metadata = {
    buildFlavor: 'release',
    licenseStatus: 'VALID',
    releaseBuild: true,
    version: '', // response for request before authentication does not reveal version
};

const metadata: Reducer<Metadata, MetadataAction> = (
    state = metadataInitialState,
    action
): Metadata => {
    switch (action.type) {
        case 'metadata/INITIAL_FETCH_METADATA_SUCCESS':
            return action.response;
        case 'metadata/POLL_METADATA_SUCCESS':
            return isEqual(state, action.response) ? state : action.response;
        default:
            return state;
    }
};

type OutdatedVersion = {
    isOutdatedVersion: boolean;
    version: string;
};

const outdatedVersionInitialState: OutdatedVersion = {
    isOutdatedVersion: false,
    version: '',
};

const outdatedVersion: Reducer<OutdatedVersion, MetadataAction> = (
    state = outdatedVersionInitialState,
    action
): OutdatedVersion => {
    switch (action.type) {
        case 'metadata/INITIAL_FETCH_METADATA_SUCCESS':
            return { ...state, version: action.response.version };
        case 'metadata/POLL_METADATA_SUCCESS': {
            if (action.response.version !== state.version) {
                return { isOutdatedVersion: true, version: action.response.version };
            }
            if (state.isOutdatedVersion) {
                return { ...state, isOutdatedVersion: false };
            }
            return state;
        }
        default:
            return state;
    }
};

const reducer = combineReducers({
    metadata,
    outdatedVersion,
});

export default reducer;

// Selectors

type State = ReturnType<typeof reducer>;

const metadataSelector = (state: State) => state.metadata;
const isOutdatedVersionSelector = (state: State) => state.outdatedVersion.isOutdatedVersion;

export const selectors = {
    metadataSelector,
    isOutdatedVersionSelector,
};
