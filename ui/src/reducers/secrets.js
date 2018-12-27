import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';
import pick from 'lodash/pick';
import { createSelector } from 'reselect';

import mergeEntitiesById from 'utils/mergeEntitiesById';
import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';
import {
    types as searchTypes,
    getActions as getSearchActions,
    reducers as searchReducers,
    getSelectors as getSearchSelectors
} from 'reducers/pageSearch';

// Action types

export const types = {
    FETCH_SECRETS: createFetchingActionTypes('secrets/FETCH_SECRETS'),
    FETCH_SECRET: createFetchingActionTypes('secrets/FETCH_SECRET'),
    ...searchTypes('secrets')
};

// Actions

export const actions = {
    fetchSecrets: createFetchingActions(types.FETCH_SECRETS),
    fetchSecret: createFetchingActions(types.FETCH_SECRET),
    ...getSearchActions('secrets')
};

// Reducers

const byId = (state = {}, action) => {
    if (action.response && action.response.entities && action.response.entities.secret) {
        const secretsById = action.response.entities.secret;
        const newState = mergeEntitiesById(state, secretsById);
        if (
            action.type === types.FETCH_SECRETS.SUCCESS &&
            (!action.params || !action.params.options || action.params.options.length === 0)
        ) {
            // fetched all secrets without any filter/search options, leave only those secrets
            const onlyExisting = pick(newState, Object.keys(secretsById));
            return isEqual(onlyExisting, state) ? state : onlyExisting;
        }
        return newState;
    }
    return state;
};

const filteredIds = (state = [], action) => {
    if (action.type === types.FETCH_SECRETS.SUCCESS) {
        return isEqual(action.response.result, state) ? state : action.response.result;
    }
    return state;
};

const reducer = combineReducers({
    byId,
    filteredIds,
    ...searchReducers('secrets')
});

export default reducer;

// Selectors

const getSecretsById = state => state.byId;
const getSecrets = createSelector(
    [getSecretsById],
    secrets => Object.values(secrets)
);
const getFilteredIds = state => state.filteredIds;
const getSecret = (state, id) => getSecretsById(state)[id];
const getFilteredSecrets = createSelector(
    [getSecretsById, getFilteredIds],
    (secrets, ids) => ids.map(id => secrets[id])
);

export const selectors = {
    getSecretsById,
    getSecrets,
    getSecret,
    getFilteredSecrets,
    ...getSearchSelectors('secrets')
};
